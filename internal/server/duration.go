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
	Context *durationContext `json:"context"`
	Input   *durationInput   `json:"input"`
	EnvTag  *string          `json:"env_tag"`
}

type durationContext struct {
	Arch          *string          `json:"arch"`
	ArchExtra     json.RawMessage  `json:"arch_extra"`
	Singers       []durationSinger `json:"singers"`
	Mix           [][]float64      `json:"mix"`
	MixSampleRate *float64         `json:"mix_sample_rate"`
	Stream        *bool            `json:"stream"`
}

type durationSinger struct {
	ID    *string         `json:"id"`
	Extra json.RawMessage `json:"extra"`
}

type durationInput struct {
	PieceDuration *float64       `json:"piece_duration"`
	Notes         []durationNote `json:"notes"`
}

type durationNote struct {
	Position      *durationPosition      `json:"position"`
	Cent          *float64               `json:"cent"`
	Pronunciation *string                `json:"pronunciation"`
	Language      *string                `json:"language"`
	Phonemes      []durationInputPhoneme `json:"phonemes"`
}

type durationPosition struct {
	Gap      *float64 `json:"gap"`
	Duration *float64 `json:"duration"`
}

type durationInputPhoneme struct {
	Token    *string `json:"token"`
	Onset    *bool   `json:"onset"`
	Language *string `json:"language"`
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
	if err := decodeJSON(c, &request); err != nil || !request.isValid() {
		durationLogger.Error("Invalid duration request", slog.Any("error", err))
		writeBadRequest(c)
		return
	}

	arch, ok := getArchitecture(*request.Context.Arch)
	if !ok {
		writeError(c, newUnknownArchError())
		return
	}

	singers := request.singers()
	envTag := arch.GetEnvTag(request.Context.ArchExtra, singers)
	if request.EnvTag != nil && *request.EnvTag == envTag {
		if c.Request.Context().Err() == nil {
			c.Status(http.StatusNoContent)
		}
		return
	}

	input := request.apiInput()
	events, err := arch.Duration(
		c.Request.Context(),
		request.Context.ArchExtra,
		singers,
		request.Context.Mix,
		*request.Context.MixSampleRate,
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

func (r durationRequest) isValid() bool {
	if r.Context == nil ||
		r.Context.Arch == nil ||
		r.Context.ArchExtra == nil ||
		r.Context.Singers == nil ||
		len(r.Context.Singers) == 0 ||
		r.Context.Mix == nil ||
		r.Context.MixSampleRate == nil ||
		*r.Context.MixSampleRate <= 0 ||
		r.Input == nil ||
		r.Input.PieceDuration == nil ||
		*r.Input.PieceDuration < 0 ||
		r.Input.Notes == nil {
		return false
	}
	for _, singer := range r.Context.Singers {
		if singer.ID == nil || singer.Extra == nil {
			return false
		}
	}
	for _, mix := range r.Context.Mix {
		if !isValidMix(mix, len(r.Context.Singers)-1) {
			return false
		}
	}
	for _, note := range r.Input.Notes {
		if !note.isValid() {
			return false
		}
	}
	return true
}

func isValidMix(mix []float64, expectedLength int) bool {
	if mix == nil || len(mix) != expectedLength {
		return false
	}
	var sum float64
	for _, value := range mix {
		if value < 0 || value > 1 {
			return false
		}
		sum += value
	}
	return sum >= 0 && sum <= 1
}

func (n durationNote) isValid() bool {
	if n.Position == nil ||
		n.Position.Gap == nil ||
		n.Position.Duration == nil ||
		*n.Position.Gap < 0 ||
		*n.Position.Duration < 0 ||
		n.Cent == nil ||
		*n.Cent < 0 ||
		*n.Cent > 12800 ||
		n.Pronunciation == nil ||
		n.Language == nil ||
		n.Phonemes == nil {
		return false
	}
	for _, phoneme := range n.Phonemes {
		if phoneme.Token == nil || phoneme.Onset == nil || phoneme.Language == nil {
			return false
		}
	}
	return true
}

func (r durationRequest) stream() bool {
	return r.Context.Stream != nil && *r.Context.Stream
}

func (r durationRequest) singers() []api.Singer {
	singers := make([]api.Singer, 0, len(r.Context.Singers))
	for _, singer := range r.Context.Singers {
		singers = append(singers, api.Singer{
			ID:    *singer.ID,
			Extra: singer.Extra,
		})
	}
	return singers
}

func (r durationRequest) apiInput() api.DurationInput {
	notes := make([]api.DurationNote, 0, len(r.Input.Notes))
	for _, note := range r.Input.Notes {
		phonemes := make([]api.DurationInputPhoneme, 0, len(note.Phonemes))
		for _, phoneme := range note.Phonemes {
			phonemes = append(phonemes, api.DurationInputPhoneme{
				Token:    *phoneme.Token,
				Onset:    *phoneme.Onset,
				Language: *phoneme.Language,
			})
		}
		notes = append(notes, api.DurationNote{
			Position: api.NotePosition{
				Gap:      *note.Position.Gap,
				Duration: *note.Position.Duration,
			},
			Cent:          *note.Cent,
			Pronunciation: *note.Pronunciation,
			Language:      *note.Language,
			Phonemes:      phonemes,
		})
	}
	return api.DurationInput{
		PieceDuration: *r.Input.PieceDuration,
		Notes:         notes,
	}
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
	var apiError *api.Error
	if !errors.As(err, &apiError) {
		durationLogger.Error("Internal duration stream error occurred", slog.Any("error", err))
		apiError = api.NewError(api.ErrorCodeInternalError, "")
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
