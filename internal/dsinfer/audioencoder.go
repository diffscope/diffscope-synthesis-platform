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

package dsinfer

import (
	"errors"
	"unsafe"

	"diffscope-synthesis-platform/native"
)

func EncodeWAV(audioData *AudioData) ([]byte, error) {
	if audioData == nil || audioData.Handle() == 0 {
		return nil, errors.New("dsinfer: audio data is not available")
	}

	rawData := native.DSSP_EncodeWAV(audioData.Handle())
	if rawData == 0 {
		return nil, errors.New("dsinfer: encode WAV failed")
	}
	defer native.DSSP_FreeDiffSingerRawData(rawData)

	size := int(native.DSSP_GetDiffSingerRawDataSize(rawData))
	if size == 0 {
		return []byte{}, nil
	}

	data := native.DSSP_GetDiffSingerRawDataBytes(rawData)
	if data == nil {
		return nil, errors.New("dsinfer: encoded WAV data is unavailable")
	}

	result := make([]byte, size)
	copy(result, unsafe.Slice(data, size))
	return result, nil
}

func EncodeFLAC(audioData *AudioData) ([]byte, error) {
	if audioData == nil || audioData.Handle() == 0 {
		return nil, errors.New("dsinfer: audio data is not available")
	}

	rawData := native.DSSP_EncodeFLAC(audioData.Handle())
	if rawData == 0 {
		return nil, errors.New("dsinfer: encode FLAC failed")
	}
	defer native.DSSP_FreeDiffSingerRawData(rawData)

	size := int(native.DSSP_GetDiffSingerRawDataSize(rawData))
	if size == 0 {
		return []byte{}, nil
	}

	data := native.DSSP_GetDiffSingerRawDataBytes(rawData)
	if data == nil {
		return nil, errors.New("dsinfer: encoded FLAC data is unavailable")
	}

	result := make([]byte, size)
	copy(result, unsafe.Slice(data, size))
	return result, nil
}
