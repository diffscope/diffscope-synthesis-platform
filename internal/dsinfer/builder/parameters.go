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

type parameterSpec struct {
	id               string
	tag              dsinfer.ParameterTag
	min              float64
	max              float64
	transform        func(float64) float64
	inverseTransform func(float64) float64
}

var parameterSpecs = []parameterSpec{
	{
		id:  "pitch",
		tag: dsinfer.ParameterTagPitch,
		min: 0,
		max: 12800,
		transform: func(value float64) float64 {
			return value / 100
		},
		inverseTransform: func(value float64) float64 {
			return value * 100
		},
	},
	{
		id:  "expressiveness",
		tag: dsinfer.ParameterTagExpr,
		min: 0,
		max: 1000,
		transform: func(value float64) float64 {
			return value / 1000
		},
		inverseTransform: func(value float64) float64 {
			return value * 1000
		},
	},
	{
		id:  "energy",
		tag: dsinfer.ParameterTagEnergy,
		min: -96000,
		max: 0,
		transform: func(value float64) float64 {
			return value / 1000
		},
		inverseTransform: func(value float64) float64 {
			return value * 1000
		},
	},
	{
		id:  "breathiness",
		tag: dsinfer.ParameterTagBreathiness,
		min: -96000,
		max: 0,
		transform: func(value float64) float64 {
			return value / 1000
		},
		inverseTransform: func(value float64) float64 {
			return value * 1000
		},
	},
	{
		id:  "voicing",
		tag: dsinfer.ParameterTagVoicing,
		min: -96000,
		max: 0,
		transform: func(value float64) float64 {
			return value / 1000
		},
		inverseTransform: func(value float64) float64 {
			return value * 1000
		},
	},
	{
		id:  "tension",
		tag: dsinfer.ParameterTagTension,
		min: -10000,
		max: 10000,
		transform: func(value float64) float64 {
			return value / 1000
		},
		inverseTransform: func(value float64) float64 {
			return value * 1000
		},
	},
	{
		id:  "mouth_opening",
		tag: dsinfer.ParameterTagMouthOpening,
		min: 0,
		max: 1000,
		transform: func(value float64) float64 {
			return value / 1000
		},
		inverseTransform: func(value float64) float64 {
			return value * 1000
		},
	},
	{
		id:  "gender",
		tag: dsinfer.ParameterTagGender,
		min: -1000,
		max: 1000,
		transform: func(value float64) float64 {
			return value / 1000
		},
		inverseTransform: func(value float64) float64 {
			return value * 1000
		},
	},
	{
		id:  "velocity",
		tag: dsinfer.ParameterTagVelocity,
		min: -1000,
		max: 1000,
		transform: func(value float64) float64 {
			return math.Exp2(value / 1000)
		},
		inverseTransform: func(value float64) float64 {
			return math.Log2(value) * 1000
		},
	},
	{
		id:  "tone_shift",
		tag: dsinfer.ParameterTagToneShift,
		min: -1200,
		max: 1200,
		transform: func(value float64) float64 {
			return value / 100
		},
		inverseTransform: func(value float64) float64 {
			return value * 100
		},
	},
}

var (
	parameterSpecsByID  = makeParameterSpecsByID(parameterSpecs)
	parameterSpecsByTag = makeParameterSpecsByTag(parameterSpecs)
)

// BuildParameter converts public parameter input into a dsinfer parameter.
func BuildParameter(
	id string,
	sampleRate float64,
	values []float64,
	retake bool,
	retakePosition int,
	retakeLength int,
) (dsinfer.Parameter, error) {
	spec, ok := parameterSpecsByID[id]
	if !ok {
		return dsinfer.Parameter{}, fmt.Errorf("dsinfer/builder: unknown parameter %q", id)
	}
	if sampleRate <= 0 || math.IsNaN(sampleRate) || math.IsInf(sampleRate, 0) {
		return dsinfer.Parameter{}, fmt.Errorf("dsinfer/builder: parameter %q sample rate must be positive and finite", id)
	}

	mappedValues := make([]float64, len(values))
	for index, value := range values {
		if math.IsNaN(value) || value < spec.min || value > spec.max {
			return dsinfer.Parameter{}, fmt.Errorf(
				"dsinfer/builder: parameter %q value %d out of range: %v not in [%v, %v]",
				id,
				index,
				value,
				spec.min,
				spec.max,
			)
		}
		mappedValues[index] = spec.transform(value)
	}

	parameter := dsinfer.Parameter{
		Tag:      spec.tag,
		Values:   mappedValues,
		Interval: 1 / sampleRate,
	}
	if retake {
		if retakePosition < 0 {
			return dsinfer.Parameter{}, fmt.Errorf("dsinfer/builder: parameter %q retake position cannot be negative", id)
		}
		if retakeLength < 0 {
			return dsinfer.Parameter{}, fmt.Errorf("dsinfer/builder: parameter %q retake length cannot be negative", id)
		}
		parameter.Retake = &dsinfer.ParameterRetake{
			Start:  float64(retakePosition) / sampleRate,
			Length: float64(retakeLength) / sampleRate,
		}
	}
	return parameter, nil
}

// ParseParameter converts a dsinfer parameter back into service-facing values.
func ParseParameter(tag dsinfer.ParameterTag, values []float64) (string, []float64, error) {
	spec, ok := parameterSpecsByTag[tag]
	if !ok {
		return "", nil, fmt.Errorf("dsinfer/builder: unknown parameter tag %d", tag)
	}

	parsedValues := make([]float64, len(values))
	for index, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return "", nil, fmt.Errorf(
				"dsinfer/builder: parameter %q value %d must be finite",
				spec.id,
				index,
			)
		}
		if spec.tag == dsinfer.ParameterTagVelocity && value <= 0 {
			return "", nil, fmt.Errorf(
				"dsinfer/builder: parameter %q value %d must be positive",
				spec.id,
				index,
			)
		}
		parsedValues[index] = spec.inverseTransform(value)
	}
	return spec.id, parsedValues, nil
}

func makeParameterSpecsByID(specs []parameterSpec) map[string]parameterSpec {
	result := make(map[string]parameterSpec, len(specs))
	for _, spec := range specs {
		result[spec.id] = spec
	}
	return result
}

func makeParameterSpecsByTag(specs []parameterSpec) map[dsinfer.ParameterTag]parameterSpec {
	result := make(map[dsinfer.ParameterTag]parameterSpec, len(specs))
	for _, spec := range specs {
		result[spec.tag] = spec
	}
	return result
}
