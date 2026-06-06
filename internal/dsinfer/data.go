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

type Words struct {
	handle uintptr
}

type Phoneme struct {
	Token             string
	Language          string
	Start             float64
	SpeakerProportion []float64
}

type Note struct {
	Cent     int
	Duration float64
	Rest     bool
}

type Speaker struct {
	ID string
}

type Word struct {
	Phonemes []Phoneme
	Notes    []Note
	Speakers []Speaker
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

		proportion, err := NewManagedDoubleArray(value.SpeakerProportion)
		if err != nil {
			phonemes.Close()
			return nil, err
		}
		native.DSSP_SetDiffSingerPhonemeSpeakerProportion(handle, int64(index), proportion.consume())
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
	}
	return &Speakers{handle: handle}, nil
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

		speakers, err := NewSpeakers(value.Speakers)
		if err != nil {
			words.Close()
			return nil, err
		}
		native.DSSP_SetDiffSingerWordSpeakers(handle, int64(index), speakers.consume())
	}
	return words, nil
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

func (w *Words) Handle() uintptr {
	if w == nil {
		return 0
	}
	return w.handle
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
			Token:             native.DSSP_GetDiffSingerPhonemeToken(handle, nativeIndex),
			Language:          native.DSSP_GetDiffSingerPhonemeLanguage(handle, nativeIndex),
			Start:             native.DSSP_GetDiffSingerPhonemeStart(handle, nativeIndex),
			SpeakerProportion: doubleArrayValues(native.DSSP_GetDiffSingerPhonemeSpeakerProportion(handle, nativeIndex)),
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
		result[index] = Speaker{
			ID: native.DSSP_GetDiffSingerSpeakerID(handle, int64(index)),
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
			Speakers: (&Speakers{handle: native.DSSP_GetDiffSingerWordSpeakers(handle, nativeIndex)}).Values(),
		}
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

func (w *Words) Close() {
	handle := w.consume()
	if handle != 0 {
		native.DSSP_FreeDiffSingerWords(handle)
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

func (w *Words) consume() uintptr {
	if w == nil {
		return 0
	}
	handle := w.handle
	w.handle = 0
	return handle
}

func validateWord(word Word) error {
	if len(word.Speakers) == 0 {
		return fmt.Errorf("dsinfer: word speakers cannot be empty")
	}
	for index, phoneme := range word.Phonemes {
		if len(phoneme.SpeakerProportion) != len(word.Speakers) {
			return fmt.Errorf("dsinfer: phoneme %d speaker proportion count does not match speaker count", index)
		}
	}
	return nil
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
