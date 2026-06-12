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

package api

import "encoding/json"

type ArchitectureSynthesisMode string

const (
	ArchitectureSynthesisModeFull      ArchitectureSynthesisMode = "FULL"
	ArchitectureSynthesisModeTokenOnly ArchitectureSynthesisMode = "TOKEN_ONLY"
	ArchitectureSynthesisModeSkip      ArchitectureSynthesisMode = "SKIP"
)

type ArchitectureParameterType string

const (
	ArchitectureParameterTypeDirect   ArchitectureParameterType = "DIRECT"
	ArchitectureParameterTypeIndirect ArchitectureParameterType = "INDIRECT"
)

type ArchitectureMetadata struct {
	Name              string                                   `json:"name"`
	PronunciationMode ArchitectureSynthesisMode                `json:"pronunciation_mode"`
	PhonemeMode       ArchitectureSynthesisMode                `json:"phoneme_mode"`
	Parameters        map[string]ArchitectureParameterMetadata `json:"parameters"`
	AudioDependencies []string                                 `json:"audio_dependencies"`
}

type ArchitectureParameterMetadata struct {
	Type      ArchitectureParameterType `json:"type"`
	DependsOn []string                  `json:"depends_on,omitempty"`
}

func (m ArchitectureParameterMetadata) MarshalJSON() ([]byte, error) {
	type directParameterMetadata struct {
		Type ArchitectureParameterType `json:"type"`
	}
	type indirectParameterMetadata struct {
		Type      ArchitectureParameterType `json:"type"`
		DependsOn []string                  `json:"depends_on"`
	}

	if m.Type != ArchitectureParameterTypeIndirect {
		return json.Marshal(directParameterMetadata{
			Type: m.Type,
		})
	}
	dependsOn := m.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}
	return json.Marshal(indirectParameterMetadata{
		Type:      m.Type,
		DependsOn: dependsOn,
	})
}
