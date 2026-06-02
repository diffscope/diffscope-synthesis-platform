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

#include "types.h"

#include <memory>

DSSP_LanguageConversionResult DSSP_ConvertLanguage(DSSP_Lyrics lyrics) {
	// TODO: Implement actual language conversion logic
	const auto *input = getLyrics(lyrics);
	auto pronunciations = std::make_unique<Pronunciations>();
	pronunciations->reserve(input->size());
	for (const auto &lyric : *input) {
		pronunciations->push_back({lyric.text, {}, false});
	}
	return {pronunciations.release(), DSSP_LanguageConversionError_None};
}
