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
	"fmt"
	"sync"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type StartRoutine func() error

var (
	startRoutinesMu sync.Mutex
	startRoutines   []StartRoutine
)

func RegisterStartRoutine(routine StartRoutine) {
	if routine == nil {
		panic("server: nil start routine")
	}

	startRoutinesMu.Lock()
	defer startRoutinesMu.Unlock()
	startRoutines = append(startRoutines, routine)
}

func runStartRoutines() error {
	startRoutinesMu.Lock()
	routines := append([]StartRoutine(nil), startRoutines...)
	startRoutinesMu.Unlock()

	for _, routine := range routines {
		if err := routine(); err != nil {
			return err
		}
	}
	return nil
}

func StartRouter() error {
	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/api/info", GetApplicationInfo)
	router.GET("/api/arch", GetArchitectureList)
	router.GET("/api/arch/:arch_id", GetArchitecture)
	router.GET("/api/singer", GetSingerList)
	router.GET("/api/arch/:arch_id/singer", GetArchSingerList)
	router.GET("/api/arch/:arch_id/singer/:singer_id", GetArchSinger)
	router.GET("/api/arch/:arch_id/singer/:singer_id/avatar", GetArchSingerAvatar)
	router.GET("/api/arch/:arch_id/singer/:singer_id/background", GetArchSingerBackground)
	router.GET("/api/arch/:arch_id/singer/:singer_id/demo_audio", GetArchSingerDemoAudioList)
	router.POST("/api/synth/pronunciation", PostPronunciation)
	router.POST("/api/synth/phoneme", PostPhoneme)
	router.POST("/api/synth/duration", PostDuration)
	router.POST("/api/synth/parameter", PostParameter)
	router.POST("/api/synth/audio", PostAudio)

	host := viper.GetString("host")
	port := viper.GetInt("port")

	return router.Run(fmt.Sprintf("%s:%d", host, port))
}

func StartServer() error {
	if err := runStartRoutines(); err != nil {
		return err
	}
	return StartRouter()
}
