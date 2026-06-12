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
	"errors"
	"log/slog"
	"net/http"

	"diffscope-synthesis-platform/internal/api"

	"github.com/gin-gonic/gin"
)

var singerLogger = slog.With("component", "server.singer")

type singerInfoResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Arch            string   `json:"arch"`
	Languages       []string `json:"languages"`
	DefaultLanguage string   `json:"default_language"`
	Extra           any      `json:"extra"`
	DefaultExtra    any      `json:"default_extra"`
}

type singerAvatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

type singerBackgroundResponse struct {
	BackgroundURL string `json:"background_url"`
}

func GetSingerList(c *gin.Context) {
	displayLanguage := c.Query("display_language")
	response := make([]singerInfoResponse, 0)
	for _, name := range registeredArchitectureNames() {
		arch, ok := getArchitecture(name)
		if !ok {
			writeSingerError(c, newUnknownArchError())
			return
		}
		singers, err := arch.GetSingerList(displayLanguage)
		if err != nil {
			writeSingerError(c, err)
			return
		}
		for _, singer := range singers {
			response = append(response, newSingerInfoResponse(name, singer))
		}
	}
	c.JSON(http.StatusOK, response)
}

func GetArchSingerList(c *gin.Context) {
	arch, ok := getArchitecture(c.Param("arch_id"))
	if !ok {
		writeSingerError(c, newUnknownArchError())
		return
	}
	singers, err := arch.GetSingerList(c.Query("display_language"))
	if err != nil {
		writeSingerError(c, err)
		return
	}
	c.JSON(http.StatusOK, singers)
}

func GetArchSinger(c *gin.Context) {
	arch, ok := getArchitecture(c.Param("arch_id"))
	if !ok {
		writeSingerError(c, newUnknownArchError())
		return
	}
	singer, err := arch.GetSinger(c.Param("singer_id"), c.Query("display_language"))
	if err != nil {
		writeSingerError(c, err)
		return
	}
	c.JSON(http.StatusOK, singer)
}

func GetArchSingerAvatar(c *gin.Context) {
	arch, ok := getArchitecture(c.Param("arch_id"))
	if !ok {
		writeSingerError(c, newUnknownArchError())
		return
	}
	avatarURL, err := arch.GetSingerAvatar(c.Param("singer_id"), c.Query("display_language"))
	if err != nil {
		writeSingerError(c, err)
		return
	}
	c.JSON(http.StatusOK, singerAvatarResponse{AvatarURL: avatarURL})
}

func GetArchSingerBackground(c *gin.Context) {
	arch, ok := getArchitecture(c.Param("arch_id"))
	if !ok {
		writeSingerError(c, newUnknownArchError())
		return
	}
	backgroundURL, err := arch.GetSingerBackground(c.Param("singer_id"), c.Query("display_language"))
	if err != nil {
		writeSingerError(c, err)
		return
	}
	c.JSON(http.StatusOK, singerBackgroundResponse{BackgroundURL: backgroundURL})
}

func GetArchSingerDemoAudioList(c *gin.Context) {
	arch, ok := getArchitecture(c.Param("arch_id"))
	if !ok {
		writeSingerError(c, newUnknownArchError())
		return
	}
	demoAudio, err := arch.GetSingerDemoAudioList(c.Param("singer_id"), c.Query("display_language"))
	if err != nil {
		writeSingerError(c, err)
		return
	}
	c.JSON(http.StatusOK, demoAudio)
}

func newSingerInfoResponse(arch string, singer api.SingerInfo) singerInfoResponse {
	return singerInfoResponse{
		ID:              singer.ID,
		Name:            singer.Name,
		Arch:            arch,
		Languages:       singer.Languages,
		DefaultLanguage: singer.DefaultLanguage,
		Extra:           singer.Extra,
		DefaultExtra:    singer.DefaultExtra,
	}
}

func writeSingerError(c *gin.Context, err error) {
	if c.Request.Context().Err() != nil {
		return
	}
	apiError := toAPIError(err)
	if !errors.As(err, &apiError) {
		singerLogger.Error("Internal error occurred", slog.Any("error", err))
	}
	status := http.StatusUnprocessableEntity
	switch apiError.Code {
	case api.ErrorCodeUnknownArch, api.ErrorCodeSingerNotExist:
		status = http.StatusNotFound
	}
	c.JSON(status, errorResponse{
		State:   api.StateError,
		Code:    apiError.Code,
		Message: apiError.Message,
	})
}
