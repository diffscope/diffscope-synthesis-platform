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

package server

import (
	"strings"
	"sync"

	"diffscope-synthesis-platform/internal/api"
	"diffscope-synthesis-platform/internal/architecture"
)

var (
	architecturesMu          sync.RWMutex
	registeredArchitectures  = make(map[string]architecture.Architecture)
	registeredArchitectureDB = architecture.NewRegistry(registeredArchitectures)
)

func RegisterArchitecture(name string, implementation architecture.Architecture) {
	if name == "" {
		panic("server: empty architecture name")
	}
	if implementation == nil {
		panic("server: nil architecture")
	}

	architecturesMu.Lock()
	defer architecturesMu.Unlock()
	registeredArchitectures[name] = implementation
	registeredArchitectureDB = architecture.NewRegistry(registeredArchitectures)
}

func getArchitecture(name string) (architecture.Architecture, bool) {
	architecturesMu.RLock()
	defer architecturesMu.RUnlock()
	return registeredArchitectureDB.Get(name)
}

func registeredArchitectureNames() []string {
	architecturesMu.RLock()
	defer architecturesMu.RUnlock()
	return registeredArchitectureDB.Names()
}

func newUnknownArchError() error {
	return api.NewError(api.ErrorCodeUnknownArch, "supported architectures: "+strings.Join(registeredArchitectureNames(), ", "))
}
