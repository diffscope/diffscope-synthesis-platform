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
	"log/slog"
	"net/http"

	"diffscope-synthesis-platform/internal/api"

	"github.com/gin-gonic/gin"
)

var phonemeLogger = slog.With("component", "server.phoneme")

type phonemeRequest struct {
	Context *pronunciationContext `json:"context"`
	Input   *phonemeInput         `json:"input"`
	EnvTag  *string               `json:"env_tag"`
}

type phonemeInput struct {
	Notes []api.PronunciationNote `json:"notes"`
}

type phonemeResponse struct {
	State  api.State     `json:"state"`
	Output phonemeOutput `json:"output"`
	EnvTag string        `json:"env_tag"`
}

type phonemeOutput struct {
	Notes []api.PhonemeNote `json:"notes"`
}

func PostPhoneme(c *gin.Context) {
	var request phonemeRequest
	if err := decodeJSON(c, &request); err != nil || !request.isValid() {
		phonemeLogger.Error("Invalid phoneme request", slog.Any("error", err))
		writeBadRequest(c)
		return
	}

	singer := api.Singer{
		ID:    *request.Context.Singer.ID,
		Extra: request.Context.Singer.Extra,
	}
	arch, ok := getArchitecture(*request.Context.Arch)
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

	notes, err := arch.Phoneme(
		c.Request.Context(),
		request.Context.ArchExtra,
		singer,
		request.Input.Notes,
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
	c.JSON(http.StatusOK, phonemeResponse{
		State: api.StateComplete,
		Output: phonemeOutput{
			Notes: notes,
		},
		EnvTag: envTag,
	})
}

func (r phonemeRequest) isValid() bool {
	return r.Context != nil &&
		r.Context.Arch != nil &&
		r.Context.ArchExtra != nil &&
		r.Context.Singer != nil &&
		r.Context.Singer.ID != nil &&
		r.Context.Singer.Extra != nil &&
		r.Input != nil &&
		r.Input.Notes != nil
}
