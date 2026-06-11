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

type SingerRequest struct {
	ID    *string          `json:"id" validate:"required"`
	Extra *json.RawMessage `json:"extra" validate:"required"`
}

func (r SingerRequest) ToSinger() Singer {
	return Singer{
		ID:    *r.ID,
		Extra: *r.Extra,
	}
}

type Lyric struct {
	Lyric    string `json:"lyric"`
	Language string `json:"language"`
}

type LyricRequest struct {
	Lyric    *string `json:"lyric" validate:"required"`
	Language *string `json:"language" validate:"required"`
}

func (r LyricRequest) ToLyric() Lyric {
	return Lyric{
		Lyric:    *r.Lyric,
		Language: *r.Language,
	}
}

type Pronunciation struct {
	Pronunciation string   `json:"pronunciation"`
	Candidates    []string `json:"candidates"`
}

type PronunciationNote struct {
	Pronunciation string `json:"pronunciation"`
	Language      string `json:"language"`
}

type PronunciationNoteRequest struct {
	Pronunciation *string `json:"pronunciation" validate:"required"`
	Language      *string `json:"language" validate:"required"`
}

func (r PronunciationNoteRequest) ToPronunciationNote() PronunciationNote {
	return PronunciationNote{
		Pronunciation: *r.Pronunciation,
		Language:      *r.Language,
	}
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

type DurationInputRequest struct {
	PieceDuration *float64              `json:"piece_duration" validate:"required,gte=0"`
	Notes         []DurationNoteRequest `json:"notes" validate:"required,dive"`
	Mix           [][]float64           `json:"mix" validate:"required,dive,required,dive,gte=0,lte=1"`
	MixSampleRate *float64              `json:"mix_sample_rate" validate:"required,gt=0"`
}

func (r DurationInputRequest) ToDurationInput() DurationInput {
	notes := make([]DurationNote, 0, len(r.Notes))
	for _, note := range r.Notes {
		notes = append(notes, note.ToDurationNote())
	}
	return DurationInput{
		PieceDuration: *r.PieceDuration,
		Notes:         notes,
	}
}

type DurationNote struct {
	Position      NotePosition           `json:"position"`
	Cent          int                    `json:"cent"`
	Pronunciation string                 `json:"pronunciation"`
	Language      string                 `json:"language"`
	Phonemes      []DurationInputPhoneme `json:"phonemes"`
}

type DurationNoteRequest struct {
	Position      *NotePositionRequest          `json:"position" validate:"required"`
	Cent          *int                          `json:"cent" validate:"required,gte=0,lte=12800"`
	Pronunciation *string                       `json:"pronunciation" validate:"required"`
	Language      *string                       `json:"language" validate:"required"`
	Phonemes      []DurationInputPhonemeRequest `json:"phonemes" validate:"required,dive"`
}

func (r DurationNoteRequest) ToDurationNote() DurationNote {
	phonemes := make([]DurationInputPhoneme, 0, len(r.Phonemes))
	for _, phoneme := range r.Phonemes {
		phonemes = append(phonemes, phoneme.ToDurationInputPhoneme())
	}
	return DurationNote{
		Position:      r.Position.ToNotePosition(),
		Cent:          *r.Cent,
		Pronunciation: *r.Pronunciation,
		Language:      *r.Language,
		Phonemes:      phonemes,
	}
}

type NotePosition struct {
	Gap      float64 `json:"gap"`
	Duration float64 `json:"duration"`
}

type NotePositionRequest struct {
	Gap      *float64 `json:"gap" validate:"required,gte=0"`
	Duration *float64 `json:"duration" validate:"required,gte=0"`
}

func (r NotePositionRequest) ToNotePosition() NotePosition {
	return NotePosition{
		Gap:      *r.Gap,
		Duration: *r.Duration,
	}
}

type DurationInputPhoneme struct {
	Token    string `json:"token"`
	Onset    bool   `json:"onset"`
	Language string `json:"language"`
}

type DurationInputPhonemeRequest struct {
	Token    *string `json:"token" validate:"required"`
	Onset    *bool   `json:"onset" validate:"required"`
	Language *string `json:"language" validate:"required"`
}

func (r DurationInputPhonemeRequest) ToDurationInputPhoneme() DurationInputPhoneme {
	return DurationInputPhoneme{
		Token:    *r.Token,
		Onset:    *r.Onset,
		Language: *r.Language,
	}
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

type ParameterInput struct {
	PieceDuration float64              `json:"piece_duration"`
	Notes         []Note               `json:"notes"`
	Parameters    map[string]Parameter `json:"parameters"`
}

type ParameterInputRequest struct {
	PieceDuration       *float64                    `json:"piece_duration" validate:"required,gte=0"`
	Notes               []ParameterNoteRequest      `json:"notes" validate:"required,dive"`
	Mix                 [][]float64                 `json:"mix" validate:"required,dive,required,dive,gte=0,lte=1"`
	MixSampleRate       *float64                    `json:"mix_sample_rate" validate:"required,gt=0"`
	ParameterSampleRate *float64                    `json:"parameter_sample_rate" validate:"required,gt=0"`
	Parameters          map[string]ParameterRequest `json:"parameters" validate:"required,dive"`
}

func (r ParameterInputRequest) ToParameterInput() ParameterInput {
	notes := make([]Note, 0, len(r.Notes))
	for _, note := range r.Notes {
		notes = append(notes, note.ToNote())
	}
	parameters := make(map[string]Parameter, len(r.Parameters))
	for name, parameter := range r.Parameters {
		parameters[name] = parameter.ToParameter()
	}
	return ParameterInput{
		PieceDuration: *r.PieceDuration,
		Notes:         notes,
		Parameters:    parameters,
	}
}

type Note struct {
	Position      NotePosition            `json:"position"`
	Cent          int                     `json:"cent"`
	Pronunciation string                  `json:"pronunciation"`
	Language      string                  `json:"language"`
	Phonemes      []ParameterInputPhoneme `json:"phonemes"`
}

type ParameterNoteRequest struct {
	Position      *NotePositionRequest           `json:"position" validate:"required"`
	Cent          *int                           `json:"cent" validate:"required,gte=0,lte=12800"`
	Pronunciation *string                        `json:"pronunciation" validate:"required"`
	Language      *string                        `json:"language" validate:"required"`
	Phonemes      []ParameterInputPhonemeRequest `json:"phonemes" validate:"required,dive"`
}

func (r ParameterNoteRequest) ToNote() Note {
	phonemes := make([]ParameterInputPhoneme, 0, len(r.Phonemes))
	for _, phoneme := range r.Phonemes {
		phonemes = append(phonemes, phoneme.ToParameterInputPhoneme())
	}
	return Note{
		Position:      r.Position.ToNotePosition(),
		Cent:          *r.Cent,
		Pronunciation: *r.Pronunciation,
		Language:      *r.Language,
		Phonemes:      phonemes,
	}
}

type ParameterInputPhoneme struct {
	Token    string  `json:"token"`
	Onset    bool    `json:"onset"`
	Language string  `json:"language"`
	Start    float64 `json:"start"`
}

type ParameterInputPhonemeRequest struct {
	Token    *string  `json:"token" validate:"required"`
	Onset    *bool    `json:"onset" validate:"required"`
	Language *string  `json:"language" validate:"required"`
	Start    *float64 `json:"start" validate:"required"`
}

func (r ParameterInputPhonemeRequest) ToParameterInputPhoneme() ParameterInputPhoneme {
	return ParameterInputPhoneme{
		Token:    *r.Token,
		Onset:    *r.Onset,
		Language: *r.Language,
		Start:    *r.Start,
	}
}

type Parameter struct {
	Values []float64        `json:"values"`
	Retake *ParameterRetake `json:"retake"`
}

type ParameterRequest struct {
	Values []float64               `json:"values" validate:"required,dive"`
	Retake *ParameterRetakeRequest `json:"retake" validate:"omitempty"`
}

func (r ParameterRequest) ToParameter() Parameter {
	var retake *ParameterRetake
	if r.Retake != nil {
		value := r.Retake.ToParameterRetake()
		retake = &value
	}
	return Parameter{
		Values: r.Values,
		Retake: retake,
	}
}

type ParameterRetake struct {
	Position int `json:"position"`
	Length   int `json:"length"`
}

type ParameterRetakeRequest struct {
	Position *int `json:"position" validate:"required,gte=0"`
	Length   *int `json:"length" validate:"required,gte=0"`
}

func (r ParameterRetakeRequest) ToParameterRetake() ParameterRetake {
	return ParameterRetake{
		Position: *r.Position,
		Length:   *r.Length,
	}
}

type ParameterOutput struct {
	Parameters map[string][]float64 `json:"parameters"`
}

type ParameterEvent struct {
	State  State
	Output ParameterOutput
	Err    error
}

type AudioInput struct {
	PieceDuration float64                   `json:"piece_duration"`
	Notes         []Note                    `json:"notes"`
	Parameters    map[string]AudioParameter `json:"parameters"`
}

type AudioInputRequest struct {
	PieceDuration       *float64                         `json:"piece_duration" validate:"required,gte=0"`
	Notes               []ParameterNoteRequest           `json:"notes" validate:"required,dive"`
	Mix                 [][]float64                      `json:"mix" validate:"required,dive,required,dive,gte=0,lte=1"`
	MixSampleRate       *float64                         `json:"mix_sample_rate" validate:"required,gt=0"`
	ParameterSampleRate *float64                         `json:"parameter_sample_rate" validate:"required,gt=0"`
	Parameters          map[string]AudioParameterRequest `json:"parameters" validate:"required,dive"`
}

func (r AudioInputRequest) ToAudioInput() AudioInput {
	notes := make([]Note, 0, len(r.Notes))
	for _, note := range r.Notes {
		notes = append(notes, note.ToNote())
	}
	parameters := make(map[string]AudioParameter, len(r.Parameters))
	for name, parameter := range r.Parameters {
		parameters[name] = parameter.ToAudioParameter()
	}
	return AudioInput{
		PieceDuration: *r.PieceDuration,
		Notes:         notes,
		Parameters:    parameters,
	}
}

type AudioParameter struct {
	Values []float64 `json:"values"`
}

type AudioParameterRequest struct {
	Values []float64 `json:"values" validate:"dive"`
}

func (r AudioParameterRequest) ToAudioParameter() AudioParameter {
	return AudioParameter{
		Values: r.Values,
	}
}

type AudioOutput struct {
	AudioURL string `json:"audio_url"`
}

type AudioEvent struct {
	State  State
	Output AudioOutput
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
