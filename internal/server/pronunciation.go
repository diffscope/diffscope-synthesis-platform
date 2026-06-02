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
	"io"
	"net/http"

	"diffscope-synthesis-platform/internal/api"

	"github.com/gin-gonic/gin"
)

type pronunciationRequest struct {
	Context *pronunciationContext `json:"context"`
	Input   *pronunciationInput   `json:"input"`
	EnvTag  *string               `json:"env_tag"`
}

type pronunciationContext struct {
	Arch      *string              `json:"arch"`
	ArchExtra json.RawMessage      `json:"arch_extra"`
	Singer    *pronunciationSinger `json:"singer"`
}

type pronunciationSinger struct {
	ID    *string         `json:"id"`
	Extra json.RawMessage `json:"extra"`
}

type pronunciationInput struct {
	Lyrics []api.Lyric `json:"lyrics"`
}

type pronunciationResponse struct {
	State  api.State           `json:"state"`
	Output pronunciationOutput `json:"output"`
	EnvTag string              `json:"env_tag"`
}

type pronunciationOutput struct {
	Pronunciations []api.Pronunciation `json:"pronunciations"`
}

type errorResponse struct {
	State   api.State     `json:"state"`
	Code    api.ErrorCode `json:"code"`
	Message string        `json:"message"`
}

func PostPronunciation(c *gin.Context) {
	var request pronunciationRequest
	if err := decodeJSON(c, &request); err != nil || !request.isValid() {
		writeBadRequest(c)
		return
	}

	singer := api.Singer{
		ID:    *request.Context.Singer.ID,
		Extra: request.Context.Singer.Extra,
	}
	arch, ok := architectures.Get(*request.Context.Arch)
	if !ok {
		writeError(c, newUnknownArchError())
		return
	}
	envTag := arch.GetEnvTag(request.Context.ArchExtra, []api.Singer{singer})
	if request.EnvTag != nil && *request.EnvTag == envTag {
		if c.Request.Context().Err() == nil {
			c.Status(http.StatusNoContent)
		}
		return
	}

	pronunciations, err := arch.Pronunciation(
		c.Request.Context(),
		request.Context.ArchExtra,
		singer,
		request.Input.Lyrics,
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
			Pronunciations: pronunciations,
		},
		EnvTag: envTag,
	})
}

func writeBadRequest(c *gin.Context) {
	if c.Request.Context().Err() != nil {
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"message": ""})
}

func decodeJSON(c *gin.Context, value any) error {
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func (r pronunciationRequest) isValid() bool {
	return r.Context != nil &&
		r.Context.Arch != nil &&
		r.Context.ArchExtra != nil &&
		r.Context.Singer != nil &&
		r.Context.Singer.ID != nil &&
		r.Context.Singer.Extra != nil &&
		r.Input != nil &&
		r.Input.Lyrics != nil
}

func writeError(c *gin.Context, err error) {
	if c.Request.Context().Err() != nil {
		return
	}
	var apiError *api.Error
	if !errors.As(err, &apiError) {
		// TODO: Log internal errors.
		apiError = api.NewError(api.ErrorCodeInternalError, "")
	}
	c.JSON(http.StatusUnprocessableEntity, errorResponse{
		State:   api.StateError,
		Code:    apiError.Code,
		Message: apiError.Message,
	})
}
