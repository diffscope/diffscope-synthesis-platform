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

var audioLogger = slog.With("component", "server.audio")

type audioRequest struct {
	Context *durationContext       `json:"context" validate:"required"`
	Input   *api.AudioInputRequest `json:"input" validate:"required"`
	EnvTag  *string                `json:"env_tag"`
	Stream  *bool                  `json:"stream"`
}

type audioResponse struct {
	State  api.State       `json:"state"`
	Output api.AudioOutput `json:"output"`
	EnvTag string          `json:"env_tag"`
}

type audioStateResponse struct {
	State api.State `json:"state"`
}

func PostAudio(c *gin.Context) {
	var request audioRequest
	if err := decodeRequest(c, &request); err != nil {
		audioLogger.Error("Invalid audio request", slog.Any("error", err))
		writeBadRequest(c)
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

	input := request.Input.ToAudioInput()
	events, err := arch.Audio(
		c.Request.Context(),
		archExtra,
		singers,
		request.Input.Mix,
		*request.Input.MixSampleRate,
		*request.Input.ParameterSampleRate,
		input.PieceDuration,
		input.Notes,
		input.Parameters,
	)
	if err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		writeError(c, err)
		return
	}
	if events == nil {
		writeError(c, api.NewError(api.ErrorCodeInternalError, "audio stream is nil"))
		return
	}

	if request.stream() {
		writeAudioStream(c, events, envTag)
		return
	}
	writeAudioResponse(c, events, envTag)
}

func (r audioRequest) stream() bool {
	return r.Stream != nil && *r.Stream
}

func (r audioRequest) singers() []api.Singer {
	singers := make([]api.Singer, 0, len(r.Context.Singers))
	for _, singer := range r.Context.Singers {
		singers = append(singers, singer.ToSinger())
	}
	return singers
}

func writeAudioResponse(c *gin.Context, events <-chan api.AudioEvent, envTag string) {
	final, err := readAudioEvents(events)
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
	c.JSON(http.StatusOK, audioResponse{
		State:  api.StateComplete,
		Output: final.Output,
		EnvTag: envTag,
	})
}

func readAudioEvents(events <-chan api.AudioEvent) (api.AudioEvent, error) {
	var previous api.State
	for event := range events {
		if err := validateAudioTransition(previous, event.State); err != nil {
			return api.AudioEvent{}, err
		}
		switch event.State {
		case api.StateComplete:
			return event, nil
		case api.StateError:
			if event.Err != nil {
				return api.AudioEvent{}, event.Err
			}
			return api.AudioEvent{}, api.NewError(api.ErrorCodeInternalError, "")
		case api.StateQueuing, api.StateProcessing:
			previous = event.State
		default:
			return api.AudioEvent{}, invalidAudioStateError()
		}
	}
	return api.AudioEvent{}, api.NewError(api.ErrorCodeInternalError, "audio stream ended without terminal state")
}

func writeAudioStream(c *gin.Context, events <-chan api.AudioEvent, envTag string) {
	writer := c.Writer
	writer.Header().Set("Content-Type", "application/x-ndjson")
	writer.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(writer)
	var previous api.State
	for event := range events {
		if c.Request.Context().Err() != nil {
			return
		}
		if err := validateAudioTransition(previous, event.State); err != nil {
			writeAudioStreamError(encoder, writer, err)
			return
		}
		switch event.State {
		case api.StateQueuing, api.StateProcessing:
			if err := encoder.Encode(audioStateResponse{State: event.State}); err != nil {
				return
			}
			previous = event.State
		case api.StateComplete:
			if err := encoder.Encode(audioResponse{
				State:  api.StateComplete,
				Output: event.Output,
				EnvTag: envTag,
			}); err != nil {
				return
			}
			flushAudioStream(writer)
			return
		case api.StateError:
			err := event.Err
			if err == nil {
				err = api.NewError(api.ErrorCodeInternalError, "")
			}
			writeAudioStreamError(encoder, writer, err)
			return
		default:
			writeAudioStreamError(encoder, writer, invalidAudioStateError())
			return
		}
		flushAudioStream(writer)
	}
	writeAudioStreamError(
		encoder,
		writer,
		api.NewError(api.ErrorCodeInternalError, "audio stream ended without terminal state"),
	)
}

func writeAudioStreamError(encoder *json.Encoder, writer gin.ResponseWriter, err error) {
	var apiError *api.Error
	if !errors.As(err, &apiError) {
		audioLogger.Error("Internal audio stream error occurred", slog.Any("error", err))
		apiError = api.NewError(api.ErrorCodeInternalError, "")
	}
	_ = encoder.Encode(errorResponse{
		State:   api.StateError,
		Code:    apiError.Code,
		Message: apiError.Message,
	})
	flushAudioStream(writer)
}

func validateAudioTransition(previous api.State, current api.State) error {
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
	return invalidAudioStateError()
}

func invalidAudioStateError() error {
	return api.NewError(api.ErrorCodeInternalError, "invalid audio state transition")
}

func flushAudioStream(writer gin.ResponseWriter) {
	if flusher, ok := any(writer).(http.Flusher); ok {
		flusher.Flush()
	}
}
