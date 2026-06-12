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

package diffsinger

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"sort"

	"diffscope-synthesis-platform/internal/api"
	"diffscope-synthesis-platform/internal/appinfo"
)

func (Architecture) GetEnvTag(archExtra json.RawMessage, singers []api.Singer) string {
	_ = archExtra
	h := sha512.New()
	h.Write([]byte(appinfo.ApplicationSemver))
	h.Write([]byte{0})
	sortedSingerIDs := make([]string, len(singers))
	for i, singer := range singers {
		sortedSingerIDs[i] = singer.ID
	}
	sort.Strings(sortedSingerIDs)
	for _, id := range sortedSingerIDs {
		_, singer, err := getSingerByAPIID(id)
		if err != nil {
			continue
		}
		h.Write([]byte(singer.PackageHash))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
