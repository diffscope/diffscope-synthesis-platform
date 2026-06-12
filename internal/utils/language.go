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

package utils

import (
	"strings"

	xlanguage "golang.org/x/text/language"
)

func BestMatch(requested string, available []string) (string, bool) {
	if len(available) == 0 {
		return "", false
	}
	if strings.TrimSpace(requested) == "" {
		return "", false
	}

	tags := make([]xlanguage.Tag, 0, len(available))
	indexes := make([]int, 0, len(available))
	for index, code := range available {
		tag, err := xlanguage.Parse(normalizeBCP47(code))
		if err != nil {
			continue
		}
		tags = append(tags, tag)
		indexes = append(indexes, index)
	}
	if len(tags) == 0 {
		return "", false
	}

	requestedTag, err := xlanguage.Parse(normalizeBCP47(requested))
	if err != nil {
		requestedTag = xlanguage.Und
	}

	_, matchedIndex, confidence := xlanguage.NewMatcher(tags).Match(requestedTag)
	if confidence == xlanguage.No {
		return "", false
	}
	return available[indexes[matchedIndex]], true
}

func normalizeBCP47(value string) string {
	value = strings.TrimSpace(value)
	if dot := strings.IndexByte(value, '.'); dot >= 0 {
		value = value[:dot]
	}
	if at := strings.IndexByte(value, '@'); at >= 0 {
		value = value[:at]
	}
	return strings.ReplaceAll(value, "_", "-")
}