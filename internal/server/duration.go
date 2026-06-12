/**************************************************************************
 * DiffScope Synthesis Platform                                           *
 * Copyright (C) 2026 Team OpenVPI                                        *
 *                                                                        *
 * This program is free software: you can redistribute it and/or modify   *
 * it under the terms of the GNU General Public License as published by   *
 * the Free Software Foundation, either version 3 of the License, or      *
 * (at your option) any later version.                                    *
 *                                                                        *
 * This program is distributed in the hope that it will be useful,        *
 * but WITHOUT ANY WARRANTY; without even the implied warranty of         *
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 * GNU General Public License for more details.                           *
 *                                                                        *
 * You should have received a copy of the GNU General Public License      *
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. *
 **************************************************************************/

package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"diffscope-synthesis-platform/internal/api"

	"github.com/gin-gonic/gin"
)

var durationLogger = slog.With("component", "server.duration")

type durationRequest struct {
	Context *durationContext          `json:"context" validate:"required"`
	Input   *api.DurationInputRequest `json:"input" validate:"required"`
	EnvTag  *string                   `json:"env_tag"`
	Stream  *bool                     `json:"stream"`
}

type durationContext struct {
	Arch      *string             `json:"arch" validate:"required"`
	ArchExtra *json.RawMessage    `json:"arch_extra" validate:"required"`
	Singers   []api.SingerRequest `json:"singers" validate:"required,min=1,dive"`
}

type durationResponse struct {
	State  api.State          `json:"state"`
	Output api.DurationOutput `json:"output"`
	EnvTag string             `json:"env_tag"`
}

type durationStateResponse struct {
	State api.State `json:"state"`
}

func PostDuration(c *gin.Context) {
	var request durationRequest
	if err := decodeRequest(c, &request); err != nil {
		durationLogger.Error("Invalid duration request", slog.Any("error", err))
		writeBadRequest(c, err)
		return
	}

	archExtra := *request.Context.ArchExtra
	arch, ok := getArchitecture(*request.Context.Arch)
	if !ok {
		writeError(c, newUnknownArchError())
		return
	}

	singers := request.singers()
	envTag := arch.GetEnvTag(archExtra, singers)
	if request.EnvTag != nil && *request.EnvTag == envTag {
		if c.Request.Context().Err() == nil {
			c.Status(http.StatusNoContent)
		}
		return
	}

	input := request.Input.ToDurationInput()
	events, err := arch.Duration(
		c.Request.Context(),
		archExtra,
		singers,
		request.Input.Mix,
		*request.Input.MixSampleRate,
		input.PieceDuration,
		input.Notes,
	)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		writeError(c, err)
		return
	}
	if events == nil {
		writeError(c, api.NewError(api.ErrorCodeInternalError, "duration stream is nil"))
		return
	}

	if request.stream() {
		writeDurationStream(c, events, envTag, input.Notes)
		return
	}
	writeDurationResponse(c, events, envTag, input.Notes)
}

func (r durationRequest) stream() bool {
	return r.Stream != nil && *r.Stream
}

func (r durationRequest) singers() []api.Singer {
	singers := make([]api.Singer, 0, len(r.Context.Singers))
	for _, singer := range r.Context.Singers {
		singers = append(singers, singer.ToSinger())
	}
	return singers
}

func writeDurationResponse(c *gin.Context, events <-chan api.DurationEvent, envTag string, notes []api.DurationNote) {
	final, err := readDurationEvents(events, notes)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		writeError(c, err)
		return
	}
	if c.Request.Context().Err() != nil {
		return
	}
	c.JSON(http.StatusOK, durationResponse{
		State:  api.StateComplete,
		Output: final.Output,
		EnvTag: envTag,
	})
}

func readDurationEvents(events <-chan api.DurationEvent, notes []api.DurationNote) (api.DurationEvent, error) {
	var previous api.State
	for event := range events {
		if err := validateDurationTransition(previous, event.State); err != nil {
			return api.DurationEvent{}, err
		}
		switch event.State {
		case api.StateComplete:
			if !isValidDurationOutput(event.Output, notes) {
				return api.DurationEvent{}, api.NewError(api.ErrorCodeInternalError, "duration output shape does not match input")
			}
			return event, nil
		case api.StateError:
			if event.Err != nil {
				return api.DurationEvent{}, event.Err
			}
			return api.DurationEvent{}, api.NewError(api.ErrorCodeInternalError, "")
		case api.StateQueuing, api.StateProcessing:
			previous = event.State
		default:
			return api.DurationEvent{}, invalidDurationStateError()
		}
	}
	return api.DurationEvent{}, api.NewError(api.ErrorCodeInternalError, "duration stream ended without terminal state")
}

func writeDurationStream(c *gin.Context, events <-chan api.DurationEvent, envTag string, notes []api.DurationNote) {
	writer := c.Writer
	writer.Header().Set("Content-Type", "application/x-ndjson")
	writer.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(writer)
	var previous api.State
	for event := range events {
		if c.Request.Context().Err() != nil {
			return
		}
		if err := validateDurationTransition(previous, event.State); err != nil {
			writeDurationStreamError(encoder, writer, err)
			return
		}
		switch event.State {
		case api.StateQueuing, api.StateProcessing:
			if err := encoder.Encode(durationStateResponse{State: event.State}); err != nil {
				return
			}
			previous = event.State
		case api.StateComplete:
			if !isValidDurationOutput(event.Output, notes) {
				writeDurationStreamError(encoder, writer, api.NewError(api.ErrorCodeInternalError, "duration output shape does not match input"))
				return
			}
			if err := encoder.Encode(durationResponse{
				State:  api.StateComplete,
				Output: event.Output,
				EnvTag: envTag,
			}); err != nil {
				return
			}
			flushDurationStream(writer)
			return
		case api.StateError:
			err := event.Err
			if err == nil {
				err = api.NewError(api.ErrorCodeInternalError, "")
			}
			writeDurationStreamError(encoder, writer, err)
			return
		default:
			writeDurationStreamError(encoder, writer, invalidDurationStateError())
			return
		}
		flushDurationStream(writer)
	}
	writeDurationStreamError(
		encoder,
		writer,
		api.NewError(api.ErrorCodeInternalError, "duration stream ended without terminal state"),
	)
}

func writeDurationStreamError(encoder *json.Encoder, writer gin.ResponseWriter, err error) {
	apiError := toAPIError(err)
	if !errors.As(err, &apiError) {
		durationLogger.Error("Internal duration stream error occurred", slog.Any("error", err))
	}
	_ = encoder.Encode(errorResponse{
		State:   api.StateError,
		Code:    apiError.Code,
		Message: apiError.Message,
	})
	flushDurationStream(writer)
}

func validateDurationTransition(previous api.State, current api.State) error {
	switch current {
	case api.StateComplete, api.StateError:
		return nil
	case api.StateQueuing:
		if previous == "" {
			return nil
		}
	case api.StateProcessing:
		if previous == "" || previous == api.StateQueuing {
			return nil
		}
	}
	return invalidDurationStateError()
}

func invalidDurationStateError() error {
	return api.NewError(api.ErrorCodeInternalError, "invalid duration state transition")
}

func isValidDurationOutput(output api.DurationOutput, notes []api.DurationNote) bool {
	if len(output.Notes) != len(notes) {
		return false
	}
	for index, note := range notes {
		if len(output.Notes[index].Phonemes) != len(note.Phonemes) {
			return false
		}
	}
	return true
}

func flushDurationStream(writer gin.ResponseWriter) {
	if flusher, ok := any(writer).(http.Flusher); ok {
		flusher.Flush()
	}
}
