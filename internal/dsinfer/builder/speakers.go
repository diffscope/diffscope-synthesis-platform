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

package builder

import (
	"fmt"
	"math"

	"diffscope-synthesis-platform/internal/dsinfer"
)

// BuildDynamicMixedSpeakers converts frame-level speaker proportions into dsinfer dynamic speakers.
func BuildDynamicMixedSpeakers(
	speakers []string,
	sampleRate float64,
	proportions [][]float64,
) ([]dsinfer.DynamicMixedSpeaker, error) {
	if len(speakers) == 0 {
		return nil, fmt.Errorf("dsinfer/builder: speakers cannot be empty")
	}
	if sampleRate <= 0 || math.IsNaN(sampleRate) || math.IsInf(sampleRate, 0) {
		return nil, fmt.Errorf("dsinfer/builder: dynamic mixed speaker sample rate must be positive and finite")
	}

	speakerCount := len(speakers)
	interval := 1 / sampleRate
	result := make([]dsinfer.DynamicMixedSpeaker, speakerCount)
	for speakerIndex, speaker := range speakers {
		result[speakerIndex] = dsinfer.DynamicMixedSpeaker{
			ID:          speaker,
			Proportions: make([]float64, len(proportions)),
			Interval:    interval,
		}
	}

	explicitSpeakerCount := speakerCount - 1
	for frameIndex, frame := range proportions {
		if len(frame) != explicitSpeakerCount {
			return nil, fmt.Errorf(
				"dsinfer/builder: dynamic mixed speaker frame %d has %d values, want %d",
				frameIndex,
				len(frame),
				explicitSpeakerCount,
			)
		}

		var sum float64
		for speakerIndex, value := range frame {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
				return nil, fmt.Errorf(
					"dsinfer/builder: dynamic mixed speaker frame %d value %d out of range: %v not in [0, 1]",
					frameIndex,
					speakerIndex,
					value,
				)
			}
			sum += value
			result[speakerIndex].Proportions[frameIndex] = value
		}
		if sum < 0 || sum > 1 {
			return nil, fmt.Errorf(
				"dsinfer/builder: dynamic mixed speaker frame %d sum out of range: %v not in [0, 1]",
				frameIndex,
				sum,
			)
		}
		result[speakerCount-1].Proportions[frameIndex] = 1 - sum
	}
	return result, nil
}
