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

var logger = slog.With("component", "server.pronunciation")

type pronunciationRequest struct {
	Context *pronunciationContext `json:"context" validate:"required"`
	Input   *pronunciationInput   `json:"input" validate:"required"`
	EnvTag  *string               `json:"env_tag"`
}

type pronunciationContext struct {
	Arch      *string            `json:"arch" validate:"required"`
	ArchExtra *json.RawMessage   `json:"arch_extra" validate:"required"`
	Singer    *api.SingerRequest `json:"singer" validate:"required"`
}

type pronunciationInput struct {
	Notes []api.LyricRequest `json:"notes" validate:"required,dive"`
}

type pronunciationResponse struct {
	State  api.State           `json:"state"`
	Output pronunciationOutput `json:"output"`
	EnvTag string              `json:"env_tag"`
}

type pronunciationOutput struct {
	Notes []api.Pronunciation `json:"notes"`
}

type errorResponse struct {
	State   api.State     `json:"state"`
	Code    api.ErrorCode `json:"code"`
	Message string        `json:"message"`
}

func PostPronunciation(c *gin.Context) {
	var request pronunciationRequest
	if err := decodeRequest(c, &request); err != nil {
		logger.Error("Invalid pronunciation request", slog.Any("error", err))
		writeBadRequest(c, err)
		return
	}

	archExtra := *request.Context.ArchExtra
	singer := request.Context.Singer.ToSinger()
	arch, ok := getArchitecture(*request.Context.Arch)
	if !ok {
		writeError(c, newUnknownArchError())
		return
	}
	envTag := arch.GetEnvTag(archExtra, []api.Singer{singer})
	if request.EnvTag != nil && *request.EnvTag == envTag {
		if c.Request.Context().Err() == nil {
			c.Status(http.StatusNoContent)
		}
		return
	}

	pronunciations, err := arch.Pronunciation(
		c.Request.Context(),
		archExtra,
		singer,
		pronunciationLyrics(request.Input.Notes),
	)
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
	c.JSON(http.StatusOK, pronunciationResponse{
		State: api.StateComplete,
		Output: pronunciationOutput{
			Notes: pronunciations,
		},
		EnvTag: envTag,
	})
}

func pronunciationLyrics(requests []api.LyricRequest) []api.Lyric {
	lyrics := make([]api.Lyric, 0, len(requests))
	for _, request := range requests {
		lyrics = append(lyrics, request.ToLyric())
	}
	return lyrics
}

func writeError(c *gin.Context, err error) {
	if c.Request.Context().Err() != nil {
		return
	}
	apiError := toAPIError(err)
	if !errors.As(err, &apiError) {
		logger.Error("Internal error occurred", slog.Any("error", err))
	}
	c.JSON(http.StatusUnprocessableEntity, errorResponse{
		State:   api.StateError,
		Code:    apiError.Code,
		Message: apiError.Message,
	})
}

func toAPIError(err error) *api.Error {
	var apiError *api.Error
	if errors.As(err, &apiError) {
		return apiError
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	return api.NewError(api.ErrorCodeInternalError, message)
}
