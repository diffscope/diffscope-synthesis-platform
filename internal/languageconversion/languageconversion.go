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

import "diffscope-synthesis-platform/native"

type Lyric struct {
	Text     string
	Language string
}

type Pronunciation struct {
	Text       string
	Candidates []string
	IsError    bool
}

type Error int

const (
	None Error = iota
	InternalError
)

func (e Error) Error() string {
	switch e {
	case None:
		return "no error"
	case InternalError:
		return "internal error"
	default:
		return "unknown error"
	}
}

func Convert(lyrics []Lyric) ([]Pronunciation, error) {
	input := native.DSSP_AllocateLyrics(int64(len(lyrics)))
	defer native.DSSP_FreeLyrics(input)

	for index, lyric := range lyrics {
		native.DSSP_SetLyricText(input, int64(index), lyric.Text)
		native.DSSP_SetLyricLanguage(input, int64(index), lyric.Language)
	}

	result := native.DSSP_ConvertLanguage(input)
	defer native.DeleteDSSP_LanguageConversionResult(result)

	output := result.GetPronunciations()
	if output != 0 {
		defer native.DSSP_FreePronunciations(output)
	}

	if err := Error(result.GetError()); err != None {
		return nil, err
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
	return pronunciations, nil
}
