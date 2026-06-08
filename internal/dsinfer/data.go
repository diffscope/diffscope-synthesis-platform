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
	"fmt"
	"unsafe"

	"diffscope-synthesis-platform/native"
)

type ManagedDoubleArray struct {
	handle uintptr
}

type Phonemes struct {
	handle uintptr
}

type Notes struct {
	handle uintptr
}

type Speakers struct {
	handle uintptr
}

type DynamicMixedSpeakers struct {
	handle uintptr
}

type Words struct {
	handle uintptr
}

type Parameters struct {
	handle uintptr
}

type ParameterTag int

const (
	ParameterTagPitch ParameterTag = iota
	ParameterTagExpr
	ParameterTagF0
	ParameterTagToneShift
	ParameterTagEnergy
	ParameterTagBreathiness
	ParameterTagVoicing
	ParameterTagTension
	ParameterTagMouthOpening
	ParameterTagGender
	ParameterTagVelocity
)

type Phoneme struct {
	Token    string
	Language string
	Start    float64
	Speakers []Speaker
}

type Note struct {
	Cent     int
	Duration float64
	Rest     bool
}

type Speaker struct {
	ID         string
	Proportion float64
}

type DynamicMixedSpeaker struct {
	ID          string
	Proportions []float64
	Interval    float64
}

type Word struct {
	Phonemes []Phoneme
	Notes    []Note
}

type Parameter struct {
	Tag      ParameterTag
	Values   []float64
	Interval float64
	Retake   *ParameterRetake
}

type ParameterRetake struct {
	Start  float64
	Length float64
}

func NewManagedDoubleArray(values []float64) (*ManagedDoubleArray, error) {
	handle := native.DSSP_AllocateDiffSingerManagedDoubleArray(int64(len(values)))
	if handle == 0 {
		return nil, fmt.Errorf("dsinfer: allocate managed double array")
	}
	copyDoubleArray(handle, values)
	return &ManagedDoubleArray{handle: handle}, nil
}

func NewPhonemes(values []Phoneme) (*Phonemes, error) {
	handle := native.DSSP_AllocateDiffSingerPhonemes(int64(len(values)))
	if handle == 0 {
		return nil, fmt.Errorf("dsinfer: allocate phonemes")
	}
	phonemes := &Phonemes{handle: handle}
	for index, value := range values {
		native.DSSP_SetDiffSingerPhonemeToken(handle, int64(index), value.Token)
		native.DSSP_SetDiffSingerPhonemeLanguage(handle, int64(index), value.Language)
		native.DSSP_SetDiffSingerPhonemeStart(handle, int64(index), value.Start)

		speakers, err := NewSpeakers(value.Speakers)
		if err != nil {
			phonemes.Close()
			return nil, err
		}
		native.DSSP_SetDiffSingerPhonemeSpeakers(handle, int64(index), speakers.consume())
	}
	return phonemes, nil
}

func NewNotes(values []Note) (*Notes, error) {
	handle := native.DSSP_AllocateDiffSingerNotes(int64(len(values)))
	if handle == 0 {
		return nil, fmt.Errorf("dsinfer: allocate notes")
	}
	for index, value := range values {
		native.DSSP_SetDiffSingerNoteCent(handle, int64(index), value.Cent)
		native.DSSP_SetDiffSingerNoteDuration(handle, int64(index), value.Duration)
		native.DSSP_SetDiffSingerNoteRest(handle, int64(index), value.Rest)
	}
	return &Notes{handle: handle}, nil
}

func NewSpeakers(values []Speaker) (*Speakers, error) {
	handle := native.DSSP_AllocateDiffSingerSpeakers(int64(len(values)))
	if handle == 0 {
		return nil, fmt.Errorf("dsinfer: allocate speakers")
	}
	for index, value := range values {
		native.DSSP_SetDiffSingerSpeakerID(handle, int64(index), value.ID)
		native.DSSP_SetDiffSingerSpeakerProportion(handle, int64(index), value.Proportion)
	}
	return &Speakers{handle: handle}, nil
}

func NewDynamicMixedSpeakers(values []DynamicMixedSpeaker) (*DynamicMixedSpeakers, error) {
	handle := native.DSSP_AllocateDiffSingerDynamicMixedSpeakers(int64(len(values)))
	if handle == 0 {
		return nil, fmt.Errorf("dsinfer: allocate dynamic mixed speakers")
	}
	speakers := &DynamicMixedSpeakers{handle: handle}
	for index, value := range values {
		native.DSSP_SetDiffSingerDynamicMixedSpeakerID(handle, int64(index), value.ID)
		native.DSSP_SetDiffSingerDynamicMixedSpeakerInterval(handle, int64(index), value.Interval)

		proportions, err := NewManagedDoubleArray(value.Proportions)
		if err != nil {
			speakers.Close()
			return nil, err
		}
		native.DSSP_SetDiffSingerDynamicMixedSpeakerProportions(handle, int64(index), proportions.consume())
	}
	return speakers, nil
}

func NewWords(values []Word) (*Words, error) {
	handle := native.DSSP_AllocateDiffSingerWords(int64(len(values)))
	if handle == 0 {
		return nil, fmt.Errorf("dsinfer: allocate words")
	}
	words := &Words{handle: handle}
	for index, value := range values {
		if err := validateWord(value); err != nil {
			words.Close()
			return nil, err
		}

		phonemes, err := NewPhonemes(value.Phonemes)
		if err != nil {
			words.Close()
			return nil, err
		}
		native.DSSP_SetDiffSingerWordPhonemes(handle, int64(index), phonemes.consume())

		notes, err := NewNotes(value.Notes)
		if err != nil {
			words.Close()
			return nil, err
		}
		native.DSSP_SetDiffSingerWordNotes(handle, int64(index), notes.consume())

	}
	return words, nil
}

func NewParameters(values []Parameter) (*Parameters, error) {
	handle := native.DSSP_AllocateDiffSingerParameters(int64(len(values)))
	if handle == 0 {
		return nil, fmt.Errorf("dsinfer: allocate parameters")
	}
	parameters := &Parameters{handle: handle}
	for index, value := range values {
		native.DSSP_SetDiffSingerParameterTag(handle, int64(index), toNativeParameterTag(value.Tag))
		native.DSSP_SetDiffSingerParameterInterval(handle, int64(index), value.Interval)
		if value.Retake != nil {
			native.DSSP_SetDiffSingerParameterRetake(handle, int64(index), true)
			native.DSSP_SetDiffSingerParameterRetakeStart(handle, int64(index), value.Retake.Start)
			native.DSSP_SetDiffSingerParameterRetakeLength(handle, int64(index), value.Retake.Length)
		}

		valueArray, err := NewManagedDoubleArray(value.Values)
		if err != nil {
			parameters.Close()
			return nil, err
		}
		native.DSSP_SetDiffSingerParameterValues(handle, int64(index), valueArray.consume())
	}
	return parameters, nil
}

func (a *ManagedDoubleArray) Handle() uintptr {
	if a == nil {
		return 0
	}
	return a.handle
}

func (p *Phonemes) Handle() uintptr {
	if p == nil {
		return 0
	}
	return p.handle
}

func (n *Notes) Handle() uintptr {
	if n == nil {
		return 0
	}
	return n.handle
}

func (s *Speakers) Handle() uintptr {
	if s == nil {
		return 0
	}
	return s.handle
}

func (s *DynamicMixedSpeakers) Handle() uintptr {
	if s == nil {
		return 0
	}
	return s.handle
}

func (w *Words) Handle() uintptr {
	if w == nil {
		return 0
	}
	return w.handle
}

func (p *Parameters) Handle() uintptr {
	if p == nil {
		return 0
	}
	return p.handle
}

func (a *ManagedDoubleArray) Values() []float64 {
	return doubleArrayValues(a.Handle())
}

func (p *Phonemes) Values() []Phoneme {
	handle := p.Handle()
	if handle == 0 {
		return nil
	}
	count := int(native.DSSP_GetDiffSingerPhonemeCount(handle))
	result := make([]Phoneme, count)
	for index := 0; index < count; index++ {
		nativeIndex := int64(index)
		result[index] = Phoneme{
			Token:    native.DSSP_GetDiffSingerPhonemeToken(handle, nativeIndex),
			Language: native.DSSP_GetDiffSingerPhonemeLanguage(handle, nativeIndex),
			Start:    native.DSSP_GetDiffSingerPhonemeStart(handle, nativeIndex),
			Speakers: (&Speakers{handle: native.DSSP_GetDiffSingerPhonemeSpeakers(handle, nativeIndex)}).Values(),
		}
	}
	return result
}

func (n *Notes) Values() []Note {
	handle := n.Handle()
	if handle == 0 {
		return nil
	}
	count := int(native.DSSP_GetDiffSingerNoteCount(handle))
	result := make([]Note, count)
	for index := 0; index < count; index++ {
		nativeIndex := int64(index)
		result[index] = Note{
			Cent:     native.DSSP_GetDiffSingerNoteCent(handle, nativeIndex),
			Duration: native.DSSP_GetDiffSingerNoteDuration(handle, nativeIndex),
			Rest:     native.DSSP_IsDiffSingerNoteRest(handle, nativeIndex),
		}
	}
	return result
}

func (s *Speakers) Values() []Speaker {
	handle := s.Handle()
	if handle == 0 {
		return nil
	}
	count := int(native.DSSP_GetDiffSingerSpeakerCount(handle))
	result := make([]Speaker, count)
	for index := 0; index < count; index++ {
		nativeIndex := int64(index)
		result[index] = Speaker{
			ID:         native.DSSP_GetDiffSingerSpeakerID(handle, nativeIndex),
			Proportion: native.DSSP_GetDiffSingerSpeakerProportion(handle, nativeIndex),
		}
	}
	return result
}

func (s *DynamicMixedSpeakers) Values() []DynamicMixedSpeaker {
	handle := s.Handle()
	if handle == 0 {
		return nil
	}
	count := int(native.DSSP_GetDiffSingerDynamicMixedSpeakerCount(handle))
	result := make([]DynamicMixedSpeaker, count)
	for index := 0; index < count; index++ {
		nativeIndex := int64(index)
		result[index] = DynamicMixedSpeaker{
			ID: native.DSSP_GetDiffSingerDynamicMixedSpeakerID(
				handle,
				nativeIndex,
			),
			Proportions: (&ManagedDoubleArray{
				handle: native.DSSP_GetDiffSingerDynamicMixedSpeakerProportions(handle, nativeIndex),
			}).Values(),
			Interval: native.DSSP_GetDiffSingerDynamicMixedSpeakerInterval(handle, nativeIndex),
		}
	}
	return result
}

func (w *Words) Values() []Word {
	handle := w.Handle()
	if handle == 0 {
		return nil
	}
	count := int(native.DSSP_GetDiffSingerWordCount(handle))
	result := make([]Word, count)
	for index := 0; index < count; index++ {
		nativeIndex := int64(index)
		result[index] = Word{
			Phonemes: (&Phonemes{handle: native.DSSP_GetDiffSingerWordPhonemes(handle, nativeIndex)}).Values(),
			Notes:    (&Notes{handle: native.DSSP_GetDiffSingerWordNotes(handle, nativeIndex)}).Values(),
		}
	}
	return result
}

func (p *Parameters) Values() []Parameter {
	handle := p.Handle()
	if handle == 0 {
		return nil
	}
	count := int(native.DSSP_GetDiffSingerParameterCount(handle))
	result := make([]Parameter, count)
	for index := 0; index < count; index++ {
		nativeIndex := int64(index)
		parameter := Parameter{
			Tag: fromNativeParameterTag(native.DSSP_GetDiffSingerParameterTag(
				handle,
				nativeIndex,
			)),
			Values: (&ManagedDoubleArray{
				handle: native.DSSP_GetDiffSingerParameterValues(handle, nativeIndex),
			}).Values(),
			Interval: native.DSSP_GetDiffSingerParameterInterval(handle, nativeIndex),
		}
		if native.DSSP_IsDiffSingerParameterRetake(handle, nativeIndex) {
			parameter.Retake = &ParameterRetake{
				Start:  native.DSSP_GetDiffSingerParameterRetakeStart(handle, nativeIndex),
				Length: native.DSSP_GetDiffSingerParameterRetakeLength(handle, nativeIndex),
			}
		}
		result[index] = parameter
	}
	return result
}

func (a *ManagedDoubleArray) Close() {
	handle := a.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerManagedDoubleArray(handle)
	}
}

func (p *Phonemes) Close() {
	handle := p.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerPhonemes(handle)
	}
}

func (n *Notes) Close() {
	handle := n.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerNotes(handle)
	}
}

func (s *Speakers) Close() {
	handle := s.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerSpeakers(handle)
	}
}

func (s *DynamicMixedSpeakers) Close() {
	handle := s.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerDynamicMixedSpeakers(handle)
	}
}

func (w *Words) Close() {
	handle := w.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerWords(handle)
	}
}

func (p *Parameters) Close() {
	handle := p.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerParameters(handle)
	}
}

func (a *ManagedDoubleArray) consume() uintptr {
	if a == nil {
		return 0
	}
	handle := a.handle
	a.handle = 0
	return handle
}

func (p *Phonemes) consume() uintptr {
	if p == nil {
		return 0
	}
	handle := p.handle
	p.handle = 0
	return handle
}

func (n *Notes) consume() uintptr {
	if n == nil {
		return 0
	}
	handle := n.handle
	n.handle = 0
	return handle
}

func (s *Speakers) consume() uintptr {
	if s == nil {
		return 0
	}
	handle := s.handle
	s.handle = 0
	return handle
}

func (s *DynamicMixedSpeakers) consume() uintptr {
	if s == nil {
		return 0
	}
	handle := s.handle
	s.handle = 0
	return handle
}

func (w *Words) consume() uintptr {
	if w == nil {
		return 0
	}
	handle := w.handle
	w.handle = 0
	return handle
}

func (p *Parameters) consume() uintptr {
	if p == nil {
		return 0
	}
	handle := p.handle
	p.handle = 0
	return handle
}

func validateWord(word Word) error {
	for index, phoneme := range word.Phonemes {
		if len(phoneme.Speakers) == 0 {
			return fmt.Errorf("dsinfer: phoneme %d speakers cannot be empty", index)
		}
	}
	return nil
}

func toNativeParameterTag(tag ParameterTag) native.DSSP_DiffSingerParameterTag {
	switch tag {
	case ParameterTagPitch:
		return native.DSSP_DiffSingerParameterTag_Pitch
	case ParameterTagExpr:
		return native.DSSP_DiffSingerParameterTag_Expr
	case ParameterTagF0:
		return native.DSSP_DiffSingerParameterTag_F0
	case ParameterTagToneShift:
		return native.DSSP_DiffSingerParameterTag_ToneShift
	case ParameterTagEnergy:
		return native.DSSP_DiffSingerParameterTag_Energy
	case ParameterTagBreathiness:
		return native.DSSP_DiffSingerParameterTag_Breathiness
	case ParameterTagVoicing:
		return native.DSSP_DiffSingerParameterTag_Voicing
	case ParameterTagTension:
		return native.DSSP_DiffSingerParameterTag_Tension
	case ParameterTagMouthOpening:
		return native.DSSP_DiffSingerParameterTag_MouthOpening
	case ParameterTagGender:
		return native.DSSP_DiffSingerParameterTag_Gender
	case ParameterTagVelocity:
		return native.DSSP_DiffSingerParameterTag_Velocity
	default:
		return native.DSSP_DiffSingerParameterTag(tag)
	}
}

func fromNativeParameterTag(tag native.DSSP_DiffSingerParameterTag) ParameterTag {
	switch tag {
	case native.DSSP_DiffSingerParameterTag_Pitch:
		return ParameterTagPitch
	case native.DSSP_DiffSingerParameterTag_Expr:
		return ParameterTagExpr
	case native.DSSP_DiffSingerParameterTag_F0:
		return ParameterTagF0
	case native.DSSP_DiffSingerParameterTag_ToneShift:
		return ParameterTagToneShift
	case native.DSSP_DiffSingerParameterTag_Energy:
		return ParameterTagEnergy
	case native.DSSP_DiffSingerParameterTag_Breathiness:
		return ParameterTagBreathiness
	case native.DSSP_DiffSingerParameterTag_Voicing:
		return ParameterTagVoicing
	case native.DSSP_DiffSingerParameterTag_Tension:
		return ParameterTagTension
	case native.DSSP_DiffSingerParameterTag_MouthOpening:
		return ParameterTagMouthOpening
	case native.DSSP_DiffSingerParameterTag_Gender:
		return ParameterTagGender
	case native.DSSP_DiffSingerParameterTag_Velocity:
		return ParameterTagVelocity
	default:
		return ParameterTag(tag)
	}
}

func copyDoubleArray(handle uintptr, values []float64) {
	if len(values) == 0 {
		return
	}
	data := native.DSSP_GetDiffSingerManagedDoubleArrayData(handle)
	copy(unsafe.Slice(data, len(values)), values)
}

func doubleArrayValues(handle uintptr) []float64 {
	if handle == 0 {
		return nil
	}
	count := int(native.DSSP_GetDiffSingerManagedDoubleArrayCount(handle))
	if count == 0 {
		return []float64{}
	}
	data := native.DSSP_GetDiffSingerManagedDoubleArrayData(handle)
	result := make([]float64, count)
	copy(result, unsafe.Slice(data, count))
	return result
}
