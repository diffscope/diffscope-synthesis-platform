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

var parameterLogger = slog.With("component", "server.parameter")

type parameterRequest struct {
	Context *durationContext           `json:"context" validate:"required"`
	Input   *api.ParameterInputRequest `json:"input" validate:"required"`
	EnvTag  *string                    `json:"env_tag"`
	Stream  *bool                      `json:"stream"`
}

type parameterResponse struct {
	State  api.State           `json:"state"`
	Output api.ParameterOutput `json:"output"`
	EnvTag string              `json:"env_tag"`
}

type parameterOutputResponse struct {
	State  api.State           `json:"state"`
	Output api.ParameterOutput `json:"output"`
}

type parameterStateResponse struct {
	State api.State `json:"state"`
}

func PostParameter(c *gin.Context) {
	var request parameterRequest
	if err := decodeRequest(c, &request); err != nil {
		parameterLogger.Error("Invalid parameter request", slog.Any("error", err))
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

	input := request.Input.ToParameterInput()
	events, err := arch.Parameter(
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
		writeError(c, api.NewError(api.ErrorCodeInternalError, "parameter stream is nil"))
		return
	}

	if request.stream() {
		writeParameterStream(c, events, envTag)
		return
	}
	writeParameterResponse(c, events, envTag)
}

func (r parameterRequest) stream() bool {
	return r.Stream != nil && *r.Stream
}

func (r parameterRequest) singers() []api.Singer {
	singers := make([]api.Singer, 0, len(r.Context.Singers))
	for _, singer := range r.Context.Singers {
		singers = append(singers, singer.ToSinger())
	}
	return singers
}

func writeParameterResponse(c *gin.Context, events <-chan api.ParameterEvent, envTag string) {
	output, err := readParameterEvents(events)
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
	c.JSON(http.StatusOK, parameterResponse{
		State:  api.StateComplete,
		Output: output,
		EnvTag: envTag,
	})
}

func readParameterEvents(events <-chan api.ParameterEvent) (api.ParameterOutput, error) {
	output := emptyParameterOutput()
	var previous api.State
	for event := range events {
		if err := validateParameterTransition(previous, event.State); err != nil {
			return api.ParameterOutput{}, err
		}
		switch event.State {
		case api.StateComplete:
			mergeParameterOutput(&output, event.Output)
			return output, nil
		case api.StateError:
			if event.Err != nil {
				return api.ParameterOutput{}, event.Err
			}
			return api.ParameterOutput{}, api.NewError(api.ErrorCodeInternalError, "")
		case api.StateQueuing:
			previous = event.State
		case api.StateProcessing:
			mergeParameterOutput(&output, event.Output)
			previous = event.State
		default:
			return api.ParameterOutput{}, invalidParameterStateError()
		}
	}
	return api.ParameterOutput{}, api.NewError(api.ErrorCodeInternalError, "parameter stream ended without terminal state")
}

func writeParameterStream(c *gin.Context, events <-chan api.ParameterEvent, envTag string) {
	writer := c.Writer
	writer.Header().Set("Content-Type", "application/x-ndjson")
	writer.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(writer)
	var previous api.State
	for event := range events {
		if c.Request.Context().Err() != nil {
			return
		}
		if err := validateParameterTransition(previous, event.State); err != nil {
			writeParameterStreamError(encoder, writer, err)
			return
		}
		switch event.State {
		case api.StateQueuing:
			if err := encoder.Encode(parameterStateResponse{State: event.State}); err != nil {
				return
			}
			previous = event.State
		case api.StateProcessing:
			if event.Output.Parameters == nil {
				if err := encoder.Encode(parameterStateResponse{State: event.State}); err != nil {
					return
				}
			} else if err := encoder.Encode(parameterOutputResponse{
				State:  event.State,
				Output: event.Output,
			}); err != nil {
				return
			}
			previous = event.State
		case api.StateComplete:
			if err := encoder.Encode(parameterResponse{
				State:  api.StateComplete,
				Output: ensureParameterOutput(event.Output),
				EnvTag: envTag,
			}); err != nil {
				return
			}
			flushParameterStream(writer)
			return
		case api.StateError:
			err := event.Err
			if err == nil {
				err = api.NewError(api.ErrorCodeInternalError, "")
			}
			writeParameterStreamError(encoder, writer, err)
			return
		default:
			writeParameterStreamError(encoder, writer, invalidParameterStateError())
			return
		}
		flushParameterStream(writer)
	}
	writeParameterStreamError(
		encoder,
		writer,
		api.NewError(api.ErrorCodeInternalError, "parameter stream ended without terminal state"),
	)
}

func writeParameterStreamError(encoder *json.Encoder, writer gin.ResponseWriter, err error) {
	var apiError *api.Error
	if !errors.As(err, &apiError) {
		parameterLogger.Error("Internal parameter stream error occurred", slog.Any("error", err))
		apiError = api.NewError(api.ErrorCodeInternalError, "")
	}
	_ = encoder.Encode(errorResponse{
		State:   api.StateError,
		Code:    apiError.Code,
		Message: apiError.Message,
	})
	flushParameterStream(writer)
}

func validateParameterTransition(previous api.State, current api.State) error {
	switch current {
	case api.StateComplete, api.StateError:
		return nil
	case api.StateQueuing:
		if previous == "" {
			return nil
		}
	case api.StateProcessing:
		if previous == "" || previous == api.StateQueuing || previous == api.StateProcessing {
			return nil
		}
	}
	return invalidParameterStateError()
}

func invalidParameterStateError() error {
	return api.NewError(api.ErrorCodeInternalError, "invalid parameter state transition")
}

func emptyParameterOutput() api.ParameterOutput {
	return api.ParameterOutput{
		Parameters: map[string][]float64{},
	}
}

func ensureParameterOutput(output api.ParameterOutput) api.ParameterOutput {
	if output.Parameters == nil {
		return emptyParameterOutput()
	}
	return output
}

func mergeParameterOutput(target *api.ParameterOutput, source api.ParameterOutput) {
	if source.Parameters == nil {
		return
	}
	if target.Parameters == nil {
		target.Parameters = make(map[string][]float64, len(source.Parameters))
	}
	for name, values := range source.Parameters {
		target.Parameters[name] = append([]float64(nil), values...)
	}
}

func flushParameterStream(writer gin.ResponseWriter) {
	if flusher, ok := any(writer).(http.Flusher); ok {
		flusher.Flush()
	}
}
