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

package architecture

import "sort"

type Registry struct {
	architectures map[string]Architecture
	names         []string
}

func NewRegistry(architectures map[string]Architecture) Registry {
	registered := make(map[string]Architecture, len(architectures))
	names := make([]string, 0, len(architectures))
	for name, implementation := range architectures {
		registered[name] = implementation
		names = append(names, name)
	}
	sort.Strings(names)
	return Registry{
		architectures: registered,
		names:         names,
	}
}

func (r Registry) Get(name string) (Architecture, bool) {
	implementation, ok := r.architectures[name]
	return implementation, ok
}

func (r Registry) Names() []string {
	return append([]string(nil), r.names...)
}
