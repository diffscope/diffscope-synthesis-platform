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
#include <utility>

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

		const ds::ParamTag &toDsinferParameterTag(DSSP_DiffSingerParameterTag tag) {
			namespace Tags = ds::Api::Common::L1::Tags;

			switch (tag) {
			case DSSP_DiffSingerParameterTag_Pitch:
				return Tags::Pitch;
			case DSSP_DiffSingerParameterTag_Expr:
				return Tags::Expr;
			case DSSP_DiffSingerParameterTag_F0:
				return Tags::F0;
			case DSSP_DiffSingerParameterTag_ToneShift:
				return Tags::ToneShift;
			case DSSP_DiffSingerParameterTag_Energy:
				return Tags::Energy;
			case DSSP_DiffSingerParameterTag_Breathiness:
				return Tags::Breathiness;
			case DSSP_DiffSingerParameterTag_Voicing:
				return Tags::Voicing;
			case DSSP_DiffSingerParameterTag_Tension:
				return Tags::Tension;
			case DSSP_DiffSingerParameterTag_MouthOpening:
				return Tags::MouthOpening;
			case DSSP_DiffSingerParameterTag_Gender:
				return Tags::Gender;
			case DSSP_DiffSingerParameterTag_Velocity:
				return Tags::Velocity;
			default:
				throw std::invalid_argument("parameter.tag");
			}
		}

		DSSP_DiffSingerParameterTag fromDsinferParameterTag(const ds::ParamTag &tag) {
			namespace Tags = ds::Api::Common::L1::Tags;

			if (tag == Tags::Pitch) {
				return DSSP_DiffSingerParameterTag_Pitch;
			}
			if (tag == Tags::Expr) {
				return DSSP_DiffSingerParameterTag_Expr;
			}
			if (tag == Tags::F0) {
				return DSSP_DiffSingerParameterTag_F0;
			}
			if (tag == Tags::ToneShift) {
				return DSSP_DiffSingerParameterTag_ToneShift;
			}
			if (tag == Tags::Energy) {
				return DSSP_DiffSingerParameterTag_Energy;
			}
			if (tag == Tags::Breathiness) {
				return DSSP_DiffSingerParameterTag_Breathiness;
			}
			if (tag == Tags::Voicing) {
				return DSSP_DiffSingerParameterTag_Voicing;
			}
			if (tag == Tags::Tension) {
				return DSSP_DiffSingerParameterTag_Tension;
			}
			if (tag == Tags::MouthOpening) {
				return DSSP_DiffSingerParameterTag_MouthOpening;
			}
			if (tag == Tags::Gender) {
				return DSSP_DiffSingerParameterTag_Gender;
			}
			if (tag == Tags::Velocity) {
				return DSSP_DiffSingerParameterTag_Velocity;
			}
			throw std::invalid_argument("parameter.tag");
		}

		ds::Api::Common::L1::InputNoteInfo toDsinferInputNoteInfo(const DiffSingerNote &note) {
			ds::Api::Common::L1::InputNoteInfo result;
			result.key = note.cent / 100;
			result.cents = note.cent % 100;
			if (result.cents >= 50) {
				++result.key;
				result.cents -= 100;
			} else if (result.cents < -50) {
				--result.key;
				result.cents += 100;
			}
			result.duration = note.duration;
			result.is_rest = note.isRest;
			return result;
		}

		ds::Api::Common::L1::InputPhonemeInfo toDsinferInputPhonemeInfo(const DiffSingerPhoneme &phoneme) {
			if (!phoneme.speakers) {
				throw std::invalid_argument("phoneme.speakers");
			}
			if (phoneme.speakers->empty()) {
				throw std::invalid_argument("phoneme.speakers");
			}

			ds::Api::Common::L1::InputPhonemeInfo result;
			result.token = phoneme.token;
			result.language = phoneme.language;
			result.start = phoneme.start;
			result.speakers.reserve(phoneme.speakers->size());
			for (const auto &item : *phoneme.speakers) {
				ds::Api::Common::L1::InputPhonemeInfo::Speaker speaker;
				speaker.name = item.id;
				speaker.proportion = item.proportion;
				result.speakers.push_back(std::move(speaker));
			}
			return result;
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

	DiffSingerDynamicMixedSpeakers *getDiffSingerDynamicMixedSpeakers(
		DSSP_DiffSingerDynamicMixedSpeakers speakers
	) {
		return static_cast<DiffSingerDynamicMixedSpeakers *>(speakers);
	}

	DiffSingerParameters *getDiffSingerParameters(DSSP_DiffSingerParameters parameters) {
		return static_cast<DiffSingerParameters *>(parameters);
	}

	DiffSingerWords *getDiffSingerWords(DSSP_DiffSingerWords words) {
		return static_cast<DiffSingerWords *>(words);
	}

	ds::Api::Common::L1::InputWordInfo toDsinferInputWordInfo(const DiffSingerWord &word) {
		if (!word.phonemes) {
			throw std::invalid_argument("word.phonemes");
		}
		if (!word.notes) {
			throw std::invalid_argument("word.notes");
		}

		ds::Api::Common::L1::InputWordInfo result;
		result.phones.reserve(word.phonemes->size());
		for (const auto &phoneme : *word.phonemes) {
			result.phones.push_back(toDsinferInputPhonemeInfo(phoneme));
		}

		result.notes.reserve(word.notes->size());
		for (const auto &note : *word.notes) {
			result.notes.push_back(toDsinferInputNoteInfo(note));
		}
		return result;
	}

	std::vector<ds::Api::Common::L1::InputWordInfo> toDsinferInputWordInfos(const DiffSingerWords &words) {
		std::vector<ds::Api::Common::L1::InputWordInfo> result;
		result.reserve(words.size());
		for (const auto &word : words) {
			result.push_back(toDsinferInputWordInfo(word));
		}
		return result;
	}

	ds::Api::Common::L1::InputSpeakerInfo toDsinferInputSpeakerInfo(
		const DiffSingerDynamicMixedSpeaker &speaker
	) {
		if (!speaker.proportions) {
			throw std::invalid_argument("speaker.proportions");
		}

		ds::Api::Common::L1::InputSpeakerInfo result;
		result.name = speaker.id;
		result.interval = speaker.interval;
		result.proportions = *speaker.proportions;
		return result;
	}

	std::vector<ds::Api::Common::L1::InputSpeakerInfo> toDsinferInputSpeakerInfos(
		const DiffSingerDynamicMixedSpeakers &speakers
	) {
		std::vector<ds::Api::Common::L1::InputSpeakerInfo> result;
		result.reserve(speakers.size());
		for (const auto &speaker : speakers) {
			result.push_back(toDsinferInputSpeakerInfo(speaker));
		}
		return result;
	}

	ds::Api::Common::L1::InputParameterInfo toDsinferInputParameterInfo(const DiffSingerParameter &parameter) {
		if (!parameter.values) {
			throw std::invalid_argument("parameter.values");
		}

		ds::Api::Common::L1::InputParameterInfo result{toDsinferParameterTag(parameter.tag)};
		result.values = *parameter.values;
		result.interval = parameter.interval;
		if (parameter.isRetake) {
			result.retake = ds::Api::Common::L1::InputParameterInfo::RetakeRange{
				parameter.retakeStart,
				parameter.retakeStart + parameter.retakeLength,
			};
		}
		return result;
	}

	std::vector<ds::Api::Common::L1::InputParameterInfo> toDsinferInputParameterInfos(
		const DiffSingerParameters &parameters
	) {
		std::vector<ds::Api::Common::L1::InputParameterInfo> result;
		result.reserve(parameters.size());
		for (const auto &parameter : parameters) {
			result.push_back(toDsinferInputParameterInfo(parameter));
		}
		return result;
	}

	DiffSingerParameter fromDsinferInputParameterInfo(
		const ds::Api::Common::L1::InputParameterInfo &parameter
	) {
		DiffSingerParameter result;
		result.tag = fromDsinferParameterTag(parameter.tag);
		result.values = std::make_unique<DiffSingerManagedDoubleArray>(parameter.values);
		result.interval = parameter.interval;
		if (parameter.retake) {
			result.isRetake = true;
			result.retakeStart = parameter.retake->start;
			result.retakeLength = parameter.retake->end - parameter.retake->start;
		}
		return result;
	}

	DiffSingerParameters fromDsinferInputParameterInfos(
		const std::vector<ds::Api::Common::L1::InputParameterInfo> &parameters
	) {
		DiffSingerParameters result;
		result.reserve(parameters.size());
		for (const auto &parameter : parameters) {
			result.push_back(fromDsinferInputParameterInfo(parameter));
		}
		return result;
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

DSSP_DiffSingerSpeakers DSSP_GetDiffSingerPhonemeSpeakers(
	DSSP_DiffSingerPhonemes phonemes,
	size_t index
) {
	const auto *result = dssp::getDiffSingerPhonemes(phonemes);
	return result->at(index).speakers.get();
}

void DSSP_SetDiffSingerPhonemeSpeakers(
	DSSP_DiffSingerPhonemes phonemes,
	size_t index,
	DSSP_DiffSingerSpeakers speakers
) {
	auto *result = dssp::getDiffSingerPhonemes(phonemes);
	auto &phoneme = result->at(index);
	phoneme.speakers = dssp::takeDiffSingerSpeakers(speakers);
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

double DSSP_GetDiffSingerSpeakerProportion(DSSP_DiffSingerSpeakers speakers, size_t index) {
	const auto *result = dssp::getDiffSingerSpeakers(speakers);
	return result->at(index).proportion;
}

void DSSP_SetDiffSingerSpeakerProportion(DSSP_DiffSingerSpeakers speakers, size_t index, double proportion) {
	auto *result = dssp::getDiffSingerSpeakers(speakers);
	result->at(index).proportion = proportion;
}

DSSP_DiffSingerDynamicMixedSpeakers DSSP_AllocateDiffSingerDynamicMixedSpeakers(size_t count) {
	return new dssp::DiffSingerDynamicMixedSpeakers(count);
}

void DSSP_FreeDiffSingerDynamicMixedSpeakers(DSSP_DiffSingerDynamicMixedSpeakers speakers) {
	delete dssp::getDiffSingerDynamicMixedSpeakers(speakers);
}

size_t DSSP_GetDiffSingerDynamicMixedSpeakerCount(DSSP_DiffSingerDynamicMixedSpeakers speakers) {
	const auto *result = dssp::getDiffSingerDynamicMixedSpeakers(speakers);
	return result->size();
}

const char *DSSP_GetDiffSingerDynamicMixedSpeakerID(
	DSSP_DiffSingerDynamicMixedSpeakers speakers,
	size_t index
) {
	const auto *result = dssp::getDiffSingerDynamicMixedSpeakers(speakers);
	return result->at(index).id.c_str();
}

void DSSP_SetDiffSingerDynamicMixedSpeakerID(
	DSSP_DiffSingerDynamicMixedSpeakers speakers,
	size_t index,
	const char *speakerID
) {
	auto *result = dssp::getDiffSingerDynamicMixedSpeakers(speakers);
	result->at(index).id = speakerID;
}

DSSP_DiffSingerManagedDoubleArray DSSP_GetDiffSingerDynamicMixedSpeakerProportions(
	DSSP_DiffSingerDynamicMixedSpeakers speakers,
	size_t index
) {
	const auto *result = dssp::getDiffSingerDynamicMixedSpeakers(speakers);
	return result->at(index).proportions.get();
}

void DSSP_SetDiffSingerDynamicMixedSpeakerProportions(
	DSSP_DiffSingerDynamicMixedSpeakers speakers,
	size_t index,
	DSSP_DiffSingerManagedDoubleArray proportions
) {
	auto *result = dssp::getDiffSingerDynamicMixedSpeakers(speakers);
	auto &speaker = result->at(index);
	speaker.proportions = dssp::takeDiffSingerManagedDoubleArray(proportions);
}

double DSSP_GetDiffSingerDynamicMixedSpeakerInterval(
	DSSP_DiffSingerDynamicMixedSpeakers speakers,
	size_t index
) {
	const auto *result = dssp::getDiffSingerDynamicMixedSpeakers(speakers);
	return result->at(index).interval;
}

void DSSP_SetDiffSingerDynamicMixedSpeakerInterval(
	DSSP_DiffSingerDynamicMixedSpeakers speakers,
	size_t index,
	double interval
) {
	auto *result = dssp::getDiffSingerDynamicMixedSpeakers(speakers);
	result->at(index).interval = interval;
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

DSSP_DiffSingerParameters DSSP_AllocateDiffSingerParameters(size_t count) {
	return new dssp::DiffSingerParameters(count);
}

void DSSP_FreeDiffSingerParameters(DSSP_DiffSingerParameters parameters) {
	delete dssp::getDiffSingerParameters(parameters);
}

size_t DSSP_GetDiffSingerParameterCount(DSSP_DiffSingerParameters parameters) {
	const auto *result = dssp::getDiffSingerParameters(parameters);
	return result->size();
}

DSSP_DiffSingerParameterTag DSSP_GetDiffSingerParameterTag(
	DSSP_DiffSingerParameters parameters,
	size_t index
) {
	const auto *result = dssp::getDiffSingerParameters(parameters);
	return result->at(index).tag;
}

void DSSP_SetDiffSingerParameterTag(
	DSSP_DiffSingerParameters parameters,
	size_t index,
	DSSP_DiffSingerParameterTag parameterName
) {
	auto *result = dssp::getDiffSingerParameters(parameters);
	result->at(index).tag = parameterName;
}

DSSP_DiffSingerManagedDoubleArray DSSP_GetDiffSingerParameterValues(
	DSSP_DiffSingerParameters parameters,
	size_t index
) {
	const auto *result = dssp::getDiffSingerParameters(parameters);
	return result->at(index).values.get();
}

void DSSP_SetDiffSingerParameterValues(
	DSSP_DiffSingerParameters parameters,
	size_t index,
	DSSP_DiffSingerManagedDoubleArray values
) {
	auto *result = dssp::getDiffSingerParameters(parameters);
	auto &parameter = result->at(index);
	parameter.values = dssp::takeDiffSingerManagedDoubleArray(values);
}

double DSSP_GetDiffSingerParameterInterval(DSSP_DiffSingerParameters parameters, size_t index) {
	const auto *result = dssp::getDiffSingerParameters(parameters);
	return result->at(index).interval;
}

void DSSP_SetDiffSingerParameterInterval(
	DSSP_DiffSingerParameters parameters,
	size_t index,
	double interval
) {
	auto *result = dssp::getDiffSingerParameters(parameters);
	result->at(index).interval = interval;
}

bool DSSP_IsDiffSingerParameterRetake(DSSP_DiffSingerParameters parameters, size_t index) {
	const auto *result = dssp::getDiffSingerParameters(parameters);
	return result->at(index).isRetake;
}

void DSSP_SetDiffSingerParameterRetake(
	DSSP_DiffSingerParameters parameters,
	size_t index,
	bool isRetake
) {
	auto *result = dssp::getDiffSingerParameters(parameters);
	result->at(index).isRetake = isRetake;
}

double DSSP_GetDiffSingerParameterRetakeStart(DSSP_DiffSingerParameters parameters, size_t index) {
	const auto *result = dssp::getDiffSingerParameters(parameters);
	return result->at(index).retakeStart;
}

void DSSP_SetDiffSingerParameterRetakeStart(
	DSSP_DiffSingerParameters parameters,
	size_t index,
	double retakeStart
) {
	auto *result = dssp::getDiffSingerParameters(parameters);
	result->at(index).retakeStart = retakeStart;
}

double DSSP_GetDiffSingerParameterRetakeLength(DSSP_DiffSingerParameters parameters, size_t index) {
	const auto *result = dssp::getDiffSingerParameters(parameters);
	return result->at(index).retakeLength;
}

void DSSP_SetDiffSingerParameterRetakeLength(
	DSSP_DiffSingerParameters parameters,
	size_t index,
	double retakeLength
) {
	auto *result = dssp::getDiffSingerParameters(parameters);
	result->at(index).retakeLength = retakeLength;
}
