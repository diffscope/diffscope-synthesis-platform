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

#ifndef DSSP_DSINFERDATA_H
#define DSSP_DSINFERDATA_H

#include "native.h"

#include <memory>
#include <string>
#include <vector>

#include <dsinfer/Api/Inferences/Common/1/CommonApiL1.h>

namespace dssp {

	using DiffSingerManagedDoubleArray = std::vector<double>;

	struct DiffSingerPhoneme {
		std::string token;
		std::string language;
		double start = 0.0;
		std::unique_ptr<DiffSingerManagedDoubleArray> speakerProportion;
	};

	struct DiffSingerNote {
		int cent = 0;
		double duration = 0.0;
		bool isRest = false;
	};

	struct DiffSingerSpeaker {
		std::string id;
	};

	using DiffSingerPhonemes = std::vector<DiffSingerPhoneme>;
	using DiffSingerNotes = std::vector<DiffSingerNote>;
	using DiffSingerSpeakers = std::vector<DiffSingerSpeaker>;

	struct DiffSingerWord {
		std::unique_ptr<DiffSingerPhonemes> phonemes;
		std::unique_ptr<DiffSingerNotes> notes;
		std::unique_ptr<DiffSingerSpeakers> speakers;
	};

	using DiffSingerWords = std::vector<DiffSingerWord>;

	DiffSingerManagedDoubleArray *getDiffSingerManagedDoubleArray(DSSP_DiffSingerManagedDoubleArray array);
	DiffSingerPhonemes *getDiffSingerPhonemes(DSSP_DiffSingerPhonemes phonemes);
	DiffSingerNotes *getDiffSingerNotes(DSSP_DiffSingerNotes notes);
	DiffSingerSpeakers *getDiffSingerSpeakers(DSSP_DiffSingerSpeakers speakers);
	DiffSingerWords *getDiffSingerWords(DSSP_DiffSingerWords words);

	ds::Api::Common::L1::InputWordInfo toDsinferInputWordInfo(const DiffSingerWord &word);
	std::vector<ds::Api::Common::L1::InputWordInfo> toDsinferInputWordInfos(const DiffSingerWords &words);

} // namespace dssp

#endif // DSSP_DSINFERDATA_H
