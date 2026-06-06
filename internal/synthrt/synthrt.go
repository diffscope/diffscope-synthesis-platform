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

package synthrt

import (
	"errors"
	"fmt"
	"sync"

	"diffscope-synthesis-platform/internal/executionprovider"
	"diffscope-synthesis-platform/native"
)

var (
	initializeOnce sync.Once
	initializeErr  error
	initialized    bool
)

type VersionNumber struct {
	Major int
	Minor int
	Patch int
	Tweak int
}

type Package struct {
	handle uintptr
}

type Singer struct {
	handle uintptr
}

func Initialize(packagePath string, device executionprovider.Device) error {
	initializeOnce.Do(func() {
		if native.DSSP_InitializeSynthRT(packagePath, device.Handle()) {
			initialized = true
			return
		}
		initializeErr = synthRTError("synthrt: initialize")
	})
	return initializeErr
}

func GetPackage(packageDir string, packageID string, version VersionNumber) (*Package, error) {
	if !initialized {
		panic("synthrt: not initialized")
	}

	nativeVersion := newNativeVersionNumber(version)
	defer native.DeleteDSSP_SRTVersionNumber(nativeVersion)

	handle := native.DSSP_GetSRTPackage(packageDir, packageID, nativeVersion)
	if handle == 0 {
		return nil, synthRTError(fmt.Sprintf(
			"synthrt: get package %s@%d.%d.%d.%d",
			packageID,
			version.Major,
			version.Minor,
			version.Patch,
			version.Tweak,
		))
	}
	return &Package{handle: handle}, nil
}

func (p *Package) Singer(singerID string) (*Singer, error) {
	return GetSinger(p, singerID)
}

func GetSinger(p *Package, singerID string) (*Singer, error) {
	if p == nil || p.handle == 0 {
		return nil, errors.New("synthrt: package is not loaded")
	}

	handle := native.DSSP_GetSRTSinger(p.handle, singerID)
	if handle == 0 {
		return nil, synthRTError(fmt.Sprintf("synthrt: get singer %q", singerID))
	}
	return &Singer{handle: handle}, nil
}

func (p *Package) Handle() uintptr {
	if p == nil {
		return 0
	}
	return p.handle
}

func (s *Singer) Handle() uintptr {
	if s == nil {
		return 0
	}
	return s.handle
}

func newNativeVersionNumber(version VersionNumber) native.DSSP_SRTVersionNumber {
	nativeVersion := native.NewDSSP_SRTVersionNumber()
	nativeVersion.SetMajor(version.Major)
	nativeVersion.SetMinor(version.Minor)
	nativeVersion.SetPatch(version.Patch)
	nativeVersion.SetTweak(version.Tweak)
	return nativeVersion
}

func synthRTError(fallback string) error {
	message := native.DSSP_GetSynthRTErrorMessage()
	if message == "" {
		message = fallback
	}
	return errors.New(message)
}
