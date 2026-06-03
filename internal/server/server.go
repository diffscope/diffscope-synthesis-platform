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

	"diffscope-synthesis-platform/internal/executionprovider"
	"diffscope-synthesis-platform/internal/languageconversion"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func StartRouter() error {
	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/api/info", GetApplicationInfo)
	router.POST("/pronunciation", PostPronunciation)

	host := viper.GetString("host")
	port := viper.GetInt("port")

	return router.Run(fmt.Sprintf("%s:%d", host, port))
}

func StartServer() error {
	device, err := getConfiguredExecutionProviderDevice()
	if err != nil {
		return err
	}
	if err := languageconversion.Initialize(device); err != nil {
		return err
	}
	return StartRouter()
}

func getConfiguredExecutionProviderDevice() (executionprovider.Device, error) {
	providerType := viper.GetString("execution_provider.type")
	deviceIndex := viper.GetInt("execution_provider.device_index")

	provider, ok := executionprovider.ParseProvider(providerType)
	if ok {
		if device, ok := executionprovider.FindDevice(provider, deviceIndex); ok {
			return device, nil
		}
	}

	return executionprovider.Device{}, fmt.Errorf(
		"execution provider device not found: type=%s, device_index=%d",
		providerType,
		deviceIndex,
	)
}
