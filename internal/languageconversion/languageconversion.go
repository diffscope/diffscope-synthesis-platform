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

package languageconversion

import (
	"errors"
	"sync"

	"diffscope-synthesis-platform/internal/executionprovider"
	"diffscope-synthesis-platform/internal/server"
	"diffscope-synthesis-platform/native"
)

var (
	initializeOnce sync.Once
	initializeErr  error
)

type Lyric struct {
	Text     string
	Language string
}

type Pronunciation struct {
	Text       string
	Candidates []string
	IsError    bool
}

func init() {
	server.RegisterStartRoutine(func() error {
		device, err := executionprovider.ConfiguredDevice()
		if err != nil {
			return err
		}
		return Initialize(device)
	})
}

func Initialize(device executionprovider.Device) error {
	initializeOnce.Do(func() {
		if native.DSSP_InitializeLanguageConversion(device.Handle()) {
			return
		}

		message := native.DSSP_GetLanguageConversionErrorMessage()
		if message == "" {
			message = "initialize language conversion"
		}
		initializeErr = errors.New(message)
	})
	return initializeErr
}

func Convert(lyrics []Lyric) []Pronunciation {
	input := native.DSSP_AllocateLyrics(int64(len(lyrics)))
	defer native.DSSP_FreeLyrics(input)

	for index, lyric := range lyrics {
		native.DSSP_SetLyricText(input, int64(index), lyric.Text)
		native.DSSP_SetLyricLanguage(input, int64(index), lyric.Language)
	}

	output := native.DSSP_ConvertLanguage(input)
	if output != 0 {
		defer native.DSSP_FreePronunciations(output)
	}

	count := native.DSSP_GetPronunciationCount(output)
	pronunciations := make([]Pronunciation, 0, count)
	for index := int64(0); index < count; index++ {
		candidateCount := native.DSSP_GetPronunciationCandidateCount(output, index)
		candidates := make([]string, 0, candidateCount)
		for candidateIndex := int64(0); candidateIndex < candidateCount; candidateIndex++ {
			candidates = append(candidates, native.DSSP_GetPronunciationCandidate(output, index, candidateIndex))
		}

		pronunciations = append(pronunciations, Pronunciation{
			Text:       native.DSSP_GetPronunciationText(output, index),
			Candidates: candidates,
			IsError:    native.DSSP_IsPronunciationError(output, index),
		})
	}
	return pronunciations
}
