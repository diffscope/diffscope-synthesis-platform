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

namespace dssp {

	Lyrics *getLyrics(DSSP_Lyrics lyrics) {
		return static_cast<Lyrics *>(lyrics);
	}

	Pronunciations *getPronunciations(DSSP_Pronunciations pronunciations) {
		return static_cast<Pronunciations *>(pronunciations);
	}

	Phonemes *getPhonemes(DSSP_Phonemes phonemes) {
		return static_cast<Phonemes *>(phonemes);
	}

} // namespace dssp

DSSP_Lyrics DSSP_AllocateLyrics(size_t count) {
	return new dssp::Lyrics(count);
}

void DSSP_FreeLyrics(DSSP_Lyrics lyrics) {
	delete dssp::getLyrics(lyrics);
}

size_t DSSP_GetLyricCount(DSSP_Lyrics lyrics) {
	const auto *result = dssp::getLyrics(lyrics);
	return result->size();
}

void DSSP_SetLyricText(DSSP_Lyrics lyrics, size_t index, const char *text) {
	auto *result = dssp::getLyrics(lyrics);
	result->at(index).text = text;
}

void DSSP_SetLyricLanguage(DSSP_Lyrics lyrics, size_t index, const char *language) {
	auto *result = dssp::getLyrics(lyrics);
	result->at(index).language = language;
}

const char *DSSP_GetLyricText(DSSP_Lyrics lyrics, size_t index) {
	const auto *result = dssp::getLyrics(lyrics);
	return result->at(index).text.c_str();
}

const char *DSSP_GetLyricLanguage(DSSP_Lyrics lyrics, size_t index) {
	const auto *result = dssp::getLyrics(lyrics);
	return result->at(index).language.c_str();
}

void DSSP_FreePronunciations(DSSP_Pronunciations pronunciations) {
	delete dssp::getPronunciations(pronunciations);
}

size_t DSSP_GetPronunciationCount(DSSP_Pronunciations pronunciations) {
	const auto *result = dssp::getPronunciations(pronunciations);
	return result->size();
}

const char *DSSP_GetPronunciationText(DSSP_Pronunciations pronunciations, size_t index) {
	const auto *result = dssp::getPronunciations(pronunciations);
	return result->at(index).text.c_str();
}

size_t DSSP_GetPronunciationCandidateCount(DSSP_Pronunciations pronunciations, size_t index) {
	const auto *result = dssp::getPronunciations(pronunciations);
	return result->at(index).candidates.size();
}

const char *DSSP_GetPronunciationCandidate(DSSP_Pronunciations pronunciations, size_t index, size_t candidate_index) {
	const auto *result = dssp::getPronunciations(pronunciations);
	return result->at(index).candidates.at(candidate_index).c_str();
}

bool DSSP_IsPronunciationError(DSSP_Pronunciations pronunciations, size_t index) {
	const auto *result = dssp::getPronunciations(pronunciations);
	return result->at(index).isError;
}

void DSSP_FreePhonemes(DSSP_Phonemes phonemes) {
	delete dssp::getPhonemes(phonemes);
}

size_t DSSP_GetPhonemeCount(DSSP_Phonemes phonemes) {
	const auto *result = dssp::getPhonemes(phonemes);
	return result->size();
}

const char *DSSP_GetPhonemeText(DSSP_Phonemes phonemes, size_t index) {
	const auto *result = dssp::getPhonemes(phonemes);
	return result->at(index).text.c_str();
}

bool DSSP_IsPhonemeOnset(DSSP_Phonemes phonemes, size_t index) {
	const auto *result = dssp::getPhonemes(phonemes);
	return result->at(index).isOnset;
}
