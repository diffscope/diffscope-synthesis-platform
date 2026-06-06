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

package diffsinger

import (
	"context"
	"encoding/json"

	"diffscope-synthesis-platform/internal/api"
)

func (Architecture) Duration(
	ctx context.Context,
	archExtra json.RawMessage,
	singers []api.Singer,
	mix [][]float64,
	mixSampleRate float64,
	pieceDuration float64,
	notes []api.DurationNote,
) (<-chan api.DurationEvent, error) {
	_ = archExtra
	_ = mix
	_ = mixSampleRate
	_ = pieceDuration

	for _, singer := range singers {
		if _, ok := getSingerMetadata(singer); !ok {
			return nil, api.NewError(api.ErrorCodeSingerNotExist, "")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	events := make(chan api.DurationEvent, 1)
	events <- api.DurationEvent{
		State:  api.StateComplete,
		Output: makeEmptyDurationOutput(notes),
	}
	close(events)
	return events, nil
}

func makeEmptyDurationOutput(notes []api.DurationNote) api.DurationOutput {
	result := api.DurationOutput{
		Notes: make([]api.DurationOutputNote, len(notes)),
	}
	for noteIndex, note := range notes {
		result.Notes[noteIndex] = api.DurationOutputNote{
			Phonemes: make([]api.DurationOutputPhoneme, len(note.Phonemes)),
		}
	}
	return result
}
