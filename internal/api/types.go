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

type Singer struct {
	ID    string          `json:"id"`
	Extra json.RawMessage `json:"extra"`
}

type Lyric struct {
	Lyric    string `json:"lyric"`
	Language string `json:"language"`
}

type Pronunciation struct {
	Pronunciation string   `json:"pronunciation"`
	Candidates    []string `json:"candidates"`
}

type PronunciationNote struct {
	Pronunciation string `json:"pronunciation"`
	Language      string `json:"language"`
}

type Phoneme struct {
	Token string `json:"token"`
	Onset bool   `json:"onset"`
}

type PhonemeNote struct {
	Phonemes []Phoneme `json:"phonemes"`
}

type DurationInput struct {
	PieceDuration float64        `json:"piece_duration"`
	Notes         []DurationNote `json:"notes"`
}

type DurationNote struct {
	Position      NotePosition           `json:"position"`
	Cent          float64                `json:"cent"`
	Pronunciation string                 `json:"pronunciation"`
	Language      string                 `json:"language"`
	Phonemes      []DurationInputPhoneme `json:"phonemes"`
}

type NotePosition struct {
	Gap      float64 `json:"gap"`
	Duration float64 `json:"duration"`
}

type DurationInputPhoneme struct {
	Token    string `json:"token"`
	Onset    bool   `json:"onset"`
	Language string `json:"language"`
}

type DurationOutput struct {
	Notes []DurationOutputNote `json:"notes"`
}

type DurationOutputNote struct {
	Phonemes []DurationOutputPhoneme `json:"phonemes"`
}

type DurationOutputPhoneme struct {
	Start float64 `json:"start"`
}

type DurationEvent struct {
	State  State
	Output DurationOutput
	Err    error
}

type State string

const (
	StatePlanned    State = "PLANNED"
	StatePending    State = "PENDING"
	StateQueuing    State = "QUEUING"
	StateProcessing State = "PROCESSING"
	StateComplete   State = "COMPLETE"
	StateError      State = "ERROR"
)
