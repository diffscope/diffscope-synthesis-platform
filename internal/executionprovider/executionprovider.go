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

package executionprovider

import (
	"fmt"
	"strings"

	"diffscope-synthesis-platform/native"

	"github.com/spf13/viper"
)

type Provider int

const (
	CPU Provider = iota
	CUDA
	DirectML
	CoreML
)

var providers = [...]Provider{
	CPU,
	CUDA,
	DirectML,
	CoreML,
}

func (p Provider) String() string {
	switch p {
	case CPU:
		return "cpu"
	case CUDA:
		return "cuda"
	case DirectML:
		return "directml"
	case CoreML:
		return "coreml"
	default:
		return "unknown"
	}
}

func ParseProvider(value string) (Provider, bool) {
	switch strings.ToLower(value) {
	case CPU.String():
		return CPU, true
	case CUDA.String():
		return CUDA, true
	case DirectML.String():
		return DirectML, true
	case CoreML.String():
		return CoreML, true
	default:
		return CPU, false
	}
}

func (p Provider) IsAvailable() bool {
	return native.DSSP_HasExecutionProvider(native.DSSP_ExecutionProvider(p))
}

func (p Provider) Devices() []Device {
	count := native.DSSP_GetExecutionProviderDeviceCount(native.DSSP_ExecutionProvider(p))
	devices := make([]Device, 0, count)
	for index := int64(0); index < count; index++ {
		devices = append(devices, Device{
			handle: native.DSSP_GetExecutionProviderDevice(native.DSSP_ExecutionProvider(p), index),
		})
	}
	return devices
}

func Providers() []Provider {
	result := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		if provider.IsAvailable() {
			result = append(result, provider)
		}
	}
	return result
}

func FindDevice(provider Provider, index int) (Device, bool) {
	if index < 0 {
		return Device{}, false
	}

	for _, device := range provider.Devices() {
		if device.Index() == index {
			return device, true
		}
	}
	return Device{}, false
}

func init() {
	defaultDevice := DefaultDevice()
	viper.SetDefault("execution_provider.type", defaultDevice.Provider().String())
	viper.SetDefault("execution_provider.device_index", defaultDevice.Index())
}

func ConfiguredDevice() (Device, error) {
	providerType := viper.GetString("execution_provider.type")
	deviceIndex := viper.GetInt("execution_provider.device_index")

	provider, ok := ParseProvider(providerType)
	if ok {
		if device, ok := FindDevice(provider, deviceIndex); ok {
			return device, nil
		}
	}

	return Device{}, fmt.Errorf(
		"execution provider device not found: type=%s, device_index=%d",
		providerType,
		deviceIndex,
	)
}

type Device struct {
	handle uintptr
}

func DefaultDevice() Device {
	return Device{handle: native.DSSP_GetDefaultDevice()}
}

func (d Device) Handle() uintptr {
	return d.handle
}

func (d Device) Provider() Provider {
	return Provider(native.DSSP_GetDeviceExecutionProvider(d.handle))
}

func (d Device) Index() int {
	return native.DSSP_GetDeviceIndex(d.handle)
}

func (d Device) Description() string {
	return native.DSSP_GetDeviceDescription(d.handle)
}

func (d Device) ID() string {
	return native.DSSP_GetDeviceID(d.handle)
}

func (d Device) Memory() uint64 {
	return native.DSSP_GetDeviceMemory(d.handle)
}
