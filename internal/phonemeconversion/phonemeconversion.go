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

package phonemeconversion

import (
	"errors"
	"sync"

	"diffscope-synthesis-platform/internal/api"
	"diffscope-synthesis-platform/native"
)

type S2P struct {
	handle uintptr

	closeOnce sync.Once
}

type OnsetMarker struct {
	handle uintptr

	closeOnce sync.Once
}

type PhonemeBuffer struct {
	handle uintptr

	closeOnce sync.Once
}

func NewDirectS2P() (*S2P, error) {
	return newS2P(native.DSSP_NewDirectS2P())
}

func NewMapS2P(mappingFilePath string) (*S2P, error) {
	return newS2P(native.DSSP_NewMapS2P(mappingFilePath))
}

func NewDictS2P(dictionaryFilePath string) (*S2P, error) {
	return newS2P(native.DSSP_NewDictS2P(dictionaryFilePath))
}

func NewCustomS2P(luaScriptFilePath string) (*S2P, error) {
	return newS2P(native.DSSP_NewCustomS2P(luaScriptFilePath))
}

func (s *S2P) Convert(pronunciation string) (*PhonemeBuffer, error) {
	if s == nil || s.handle == 0 {
		return nil, errors.New("phonemeconversion: S2P is closed")
	}

	handle := native.DSSP_RunS2P(s.handle, pronunciation)
	if handle == 0 {
		return nil, errors.New("phonemeconversion: run S2P")
	}
	return &PhonemeBuffer{handle: handle}, nil
}

func (s *S2P) TerminateCustom() {
	if s == nil || s.handle == 0 {
		return
	}
	native.DSSP_TerminateCustomS2P(s.handle)
}

func (s *S2P) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.handle != 0 {
			native.DSSP_DeleteS2P(s.handle)
			s.handle = 0
		}
	})
}

func NewRuleOnsetMarker(ruleFilePath string) (*OnsetMarker, error) {
	return newOnsetMarker(native.DSSP_NewRuleOnsetMarker(ruleFilePath))
}

func NewCustomOnsetMarker(luaScriptFilePath string) (*OnsetMarker, error) {
	return newOnsetMarker(native.DSSP_NewCustomOnsetMarker(luaScriptFilePath))
}

func (m *OnsetMarker) Mark(phonemes *PhonemeBuffer) error {
	if m == nil || m.handle == 0 {
		return errors.New("phonemeconversion: onset marker is closed")
	}
	if phonemes == nil || phonemes.handle == 0 {
		return errors.New("phonemeconversion: phoneme buffer is closed")
	}

	native.DSSP_RunOnsetMarker(m.handle, phonemes.handle)
	return nil
}

func (m *OnsetMarker) TerminateCustom() {
	if m == nil || m.handle == 0 {
		return
	}
	native.DSSP_TerminateCustomOnsetMarker(m.handle)
}

func (m *OnsetMarker) Close() {
	if m == nil {
		return
	}
	m.closeOnce.Do(func() {
		if m.handle != 0 {
			native.DSSP_DeleteOnsetMarker(m.handle)
			m.handle = 0
		}
	})
}

func (p *PhonemeBuffer) Snapshot() []api.Phoneme {
	if p == nil || p.handle == 0 {
		return []api.Phoneme{}
	}

	count := native.DSSP_GetPhonemeCount(p.handle)
	phonemes := make([]api.Phoneme, 0, count)
	for index := int64(0); index < count; index++ {
		phonemes = append(phonemes, api.Phoneme{
			Token: native.DSSP_GetPhonemeText(p.handle, index),
			Onset: native.DSSP_IsPhonemeOnset(
				p.handle,
				index,
			),
		})
	}
	return phonemes
}

func (p *PhonemeBuffer) Close() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		if p.handle != 0 {
			native.DSSP_FreePhonemes(p.handle)
			p.handle = 0
		}
	})
}

func SetLuaRunnerCount(count int) {
	native.DSSP_SetLuaRunnerCount(int64(count))
}

func newS2P(handle uintptr) (*S2P, error) {
	if handle == 0 {
		return nil, errors.New("phonemeconversion: create S2P")
	}
	if native.DSSP_IsS2PError(handle) {
		message := native.DSSP_GetS2PErrorMessage(handle)
		native.DSSP_DeleteS2P(handle)
		if message == "" {
			message = "phonemeconversion: create S2P"
		}
		return nil, errors.New(message)
	}
	return &S2P{handle: handle}, nil
}

func newOnsetMarker(handle uintptr) (*OnsetMarker, error) {
	if handle == 0 {
		return nil, errors.New("phonemeconversion: create onset marker")
	}
	if native.DSSP_IsOnsetMarkerError(handle) {
		message := native.DSSP_GetOnsetMarkerErrorMessage(handle)
		native.DSSP_DeleteOnsetMarker(handle)
		if message == "" {
			message = "phonemeconversion: create onset marker"
		}
		return nil, errors.New(message)
	}
	return &OnsetMarker{handle: handle}, nil
}
