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

#include "dsinferdata.h"

#include <memory>
#include <stdexcept>

namespace dssp {

	namespace {

		std::unique_ptr<DiffSingerManagedDoubleArray> takeDiffSingerManagedDoubleArray(
			DSSP_DiffSingerManagedDoubleArray array
		) {
			auto *result = getDiffSingerManagedDoubleArray(array);
			if (result == nullptr) {
				throw std::invalid_argument("array");
			}
			return std::unique_ptr<DiffSingerManagedDoubleArray>(result);
		}

		std::unique_ptr<DiffSingerPhonemes> takeDiffSingerPhonemes(DSSP_DiffSingerPhonemes phonemes) {
			auto *result = getDiffSingerPhonemes(phonemes);
			if (result == nullptr) {
				throw std::invalid_argument("phonemes");
			}
			return std::unique_ptr<DiffSingerPhonemes>(result);
		}

		std::unique_ptr<DiffSingerNotes> takeDiffSingerNotes(DSSP_DiffSingerNotes notes) {
			auto *result = getDiffSingerNotes(notes);
			if (result == nullptr) {
				throw std::invalid_argument("notes");
			}
			return std::unique_ptr<DiffSingerNotes>(result);
		}

		std::unique_ptr<DiffSingerSpeakers> takeDiffSingerSpeakers(DSSP_DiffSingerSpeakers speakers) {
			auto *result = getDiffSingerSpeakers(speakers);
			if (result == nullptr) {
				throw std::invalid_argument("speakers");
			}
			return std::unique_ptr<DiffSingerSpeakers>(result);
		}

	} // namespace

	DiffSingerManagedDoubleArray *getDiffSingerManagedDoubleArray(DSSP_DiffSingerManagedDoubleArray array) {
		return static_cast<DiffSingerManagedDoubleArray *>(array);
	}

	DiffSingerPhonemes *getDiffSingerPhonemes(DSSP_DiffSingerPhonemes phonemes) {
		return static_cast<DiffSingerPhonemes *>(phonemes);
	}

	DiffSingerNotes *getDiffSingerNotes(DSSP_DiffSingerNotes notes) {
		return static_cast<DiffSingerNotes *>(notes);
	}

	DiffSingerSpeakers *getDiffSingerSpeakers(DSSP_DiffSingerSpeakers speakers) {
		return static_cast<DiffSingerSpeakers *>(speakers);
	}

	DiffSingerWords *getDiffSingerWords(DSSP_DiffSingerWords words) {
		return static_cast<DiffSingerWords *>(words);
	}

} // namespace dssp

DSSP_DiffSingerManagedDoubleArray DSSP_AllocateDiffSingerManagedDoubleArray(size_t count) {
	return new dssp::DiffSingerManagedDoubleArray(count);
}

void DSSP_FreeDiffSingerManagedDoubleArray(DSSP_DiffSingerManagedDoubleArray array) {
	delete dssp::getDiffSingerManagedDoubleArray(array);
}

size_t DSSP_GetDiffSingerManagedDoubleArrayCount(DSSP_DiffSingerManagedDoubleArray array) {
	const auto *result = dssp::getDiffSingerManagedDoubleArray(array);
	return result->size();
}

double *DSSP_GetDiffSingerManagedDoubleArrayData(DSSP_DiffSingerManagedDoubleArray array) {
	auto *result = dssp::getDiffSingerManagedDoubleArray(array);
	return result->data();
}

DSSP_DiffSingerPhonemes DSSP_AllocateDiffSingerPhonemes(size_t count) {
	return new dssp::DiffSingerPhonemes(count);
}

void DSSP_FreeDiffSingerPhonemes(DSSP_DiffSingerPhonemes phonemes) {
	delete dssp::getDiffSingerPhonemes(phonemes);
}

size_t DSSP_GetDiffSingerPhonemeCount(DSSP_DiffSingerPhonemes phonemes) {
	const auto *result = dssp::getDiffSingerPhonemes(phonemes);
	return result->size();
}

const char *DSSP_GetDiffSingerPhonemeToken(DSSP_DiffSingerPhonemes phonemes, size_t index) {
	const auto *result = dssp::getDiffSingerPhonemes(phonemes);
	return result->at(index).token.c_str();
}

void DSSP_SetDiffSingerPhonemeToken(DSSP_DiffSingerPhonemes phonemes, size_t index, const char *token) {
	auto *result = dssp::getDiffSingerPhonemes(phonemes);
	result->at(index).token = token;
}

const char *DSSP_GetDiffSingerPhonemeLanguage(DSSP_DiffSingerPhonemes phonemes, size_t index) {
	const auto *result = dssp::getDiffSingerPhonemes(phonemes);
	return result->at(index).language.c_str();
}

void DSSP_SetDiffSingerPhonemeLanguage(DSSP_DiffSingerPhonemes phonemes, size_t index, const char *language) {
	auto *result = dssp::getDiffSingerPhonemes(phonemes);
	result->at(index).language = language;
}

double DSSP_GetDiffSingerPhonemeStart(DSSP_DiffSingerPhonemes phonemes, size_t index) {
	const auto *result = dssp::getDiffSingerPhonemes(phonemes);
	return result->at(index).start;
}

void DSSP_SetDiffSingerPhonemeStart(DSSP_DiffSingerPhonemes phonemes, size_t index, double start) {
	auto *result = dssp::getDiffSingerPhonemes(phonemes);
	result->at(index).start = start;
}

DSSP_DiffSingerManagedDoubleArray DSSP_GetDiffSingerPhonemeSpeakerProportion(
	DSSP_DiffSingerPhonemes phonemes,
	size_t index
) {
	const auto *result = dssp::getDiffSingerPhonemes(phonemes);
	return result->at(index).speakerProportion.get();
}

void DSSP_SetDiffSingerPhonemeSpeakerProportion(
	DSSP_DiffSingerPhonemes phonemes,
	size_t index,
	DSSP_DiffSingerManagedDoubleArray speakerProportion
) {
	auto *result = dssp::getDiffSingerPhonemes(phonemes);
	auto &phoneme = result->at(index);
	phoneme.speakerProportion = dssp::takeDiffSingerManagedDoubleArray(speakerProportion);
}

DSSP_DiffSingerNotes DSSP_AllocateDiffSingerNotes(size_t count) {
	return new dssp::DiffSingerNotes(count);
}

void DSSP_FreeDiffSingerNotes(DSSP_DiffSingerNotes notes) {
	delete dssp::getDiffSingerNotes(notes);
}

size_t DSSP_GetDiffSingerNoteCount(DSSP_DiffSingerNotes notes) {
	const auto *result = dssp::getDiffSingerNotes(notes);
	return result->size();
}

int DSSP_GetDiffSingerNoteCent(DSSP_DiffSingerNotes notes, size_t index) {
	const auto *result = dssp::getDiffSingerNotes(notes);
	return result->at(index).cent;
}

void DSSP_SetDiffSingerNoteCent(DSSP_DiffSingerNotes notes, size_t index, int cent) {
	auto *result = dssp::getDiffSingerNotes(notes);
	result->at(index).cent = cent;
}

double DSSP_GetDiffSingerNoteDuration(DSSP_DiffSingerNotes notes, size_t index) {
	const auto *result = dssp::getDiffSingerNotes(notes);
	return result->at(index).duration;
}

void DSSP_SetDiffSingerNoteDuration(DSSP_DiffSingerNotes notes, size_t index, double duration) {
	auto *result = dssp::getDiffSingerNotes(notes);
	result->at(index).duration = duration;
}

bool DSSP_IsDiffSingerNoteRest(DSSP_DiffSingerNotes notes, size_t index) {
	const auto *result = dssp::getDiffSingerNotes(notes);
	return result->at(index).isRest;
}

void DSSP_SetDiffSingerNoteRest(DSSP_DiffSingerNotes notes, size_t index, bool isRest) {
	auto *result = dssp::getDiffSingerNotes(notes);
	result->at(index).isRest = isRest;
}

DSSP_DiffSingerSpeakers DSSP_AllocateDiffSingerSpeakers(size_t count) {
	return new dssp::DiffSingerSpeakers(count);
}

void DSSP_FreeDiffSingerSpeakers(DSSP_DiffSingerSpeakers speakers) {
	delete dssp::getDiffSingerSpeakers(speakers);
}

size_t DSSP_GetDiffSingerSpeakerCount(DSSP_DiffSingerSpeakers speakers) {
	const auto *result = dssp::getDiffSingerSpeakers(speakers);
	return result->size();
}

const char *DSSP_GetDiffSingerSpeakerID(DSSP_DiffSingerSpeakers speakers, size_t index) {
	const auto *result = dssp::getDiffSingerSpeakers(speakers);
	return result->at(index).id.c_str();
}

void DSSP_SetDiffSingerSpeakerID(DSSP_DiffSingerSpeakers speakers, size_t index, const char *speakerID) {
	auto *result = dssp::getDiffSingerSpeakers(speakers);
	result->at(index).id = speakerID;
}

DSSP_DiffSingerWords DSSP_AllocateDiffSingerWords(size_t count) {
	return new dssp::DiffSingerWords(count);
}

void DSSP_FreeDiffSingerWords(DSSP_DiffSingerWords words) {
	delete dssp::getDiffSingerWords(words);
}

size_t DSSP_GetDiffSingerWordCount(DSSP_DiffSingerWords words) {
	const auto *result = dssp::getDiffSingerWords(words);
	return result->size();
}

DSSP_DiffSingerPhonemes DSSP_GetDiffSingerWordPhonemes(DSSP_DiffSingerWords words, size_t index) {
	const auto *result = dssp::getDiffSingerWords(words);
	return result->at(index).phonemes.get();
}

void DSSP_SetDiffSingerWordPhonemes(DSSP_DiffSingerWords words, size_t index, DSSP_DiffSingerPhonemes phonemes) {
	auto *result = dssp::getDiffSingerWords(words);
	auto &word = result->at(index);
	word.phonemes = dssp::takeDiffSingerPhonemes(phonemes);
}

DSSP_DiffSingerNotes DSSP_GetDiffSingerWordNotes(DSSP_DiffSingerWords words, size_t index) {
	const auto *result = dssp::getDiffSingerWords(words);
	return result->at(index).notes.get();
}

void DSSP_SetDiffSingerWordNotes(DSSP_DiffSingerWords words, size_t index, DSSP_DiffSingerNotes notes) {
	auto *result = dssp::getDiffSingerWords(words);
	auto &word = result->at(index);
	word.notes = dssp::takeDiffSingerNotes(notes);
}

DSSP_DiffSingerSpeakers DSSP_GetDiffSingerWordSpeakers(DSSP_DiffSingerWords words, size_t index) {
	const auto *result = dssp::getDiffSingerWords(words);
	return result->at(index).speakers.get();
}

void DSSP_SetDiffSingerWordSpeakers(DSSP_DiffSingerWords words, size_t index, DSSP_DiffSingerSpeakers speakers) {
	auto *result = dssp::getDiffSingerWords(words);
	auto &word = result->at(index);
	word.speakers = dssp::takeDiffSingerSpeakers(speakers);
}
