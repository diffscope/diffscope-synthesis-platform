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

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

#include <dsinfer/Api/Inferences/Common/1/CommonApiL1.h>
#include <dsinfer/Core/Tensor.h>

namespace dssp {

	using DiffSingerManagedDoubleArray = std::vector<double>;

	struct DiffSingerSpeaker {
		std::string id;
		double proportion = 0.0;
	};

	using DiffSingerSpeakers = std::vector<DiffSingerSpeaker>;

	struct DiffSingerDynamicMixedSpeaker {
		std::string id;
		std::unique_ptr<DiffSingerManagedDoubleArray> proportions;
		double interval = 0.0;
	};

	using DiffSingerDynamicMixedSpeakers = std::vector<DiffSingerDynamicMixedSpeaker>;

	struct DiffSingerParameter {
		DSSP_DiffSingerParameterTag tag = DSSP_DiffSingerParameterTag_Pitch;
		std::unique_ptr<DiffSingerManagedDoubleArray> values;
		double interval = 0.0;
		bool isRetake = false;
		double retakeStart = 0.0;
		double retakeLength = 0.0;
	};

	using DiffSingerParameters = std::vector<DiffSingerParameter>;

	struct DiffSingerPhoneme {
		std::string token;
		std::string language;
		double start = 0.0;
		std::unique_ptr<DiffSingerSpeakers> speakers;
	};

	struct DiffSingerNote {
		int cent = 0;
		double duration = 0.0;
		bool isRest = false;
	};

	using DiffSingerPhonemes = std::vector<DiffSingerPhoneme>;
	using DiffSingerNotes = std::vector<DiffSingerNote>;

	struct DiffSingerWord {
		std::unique_ptr<DiffSingerPhonemes> phonemes;
		std::unique_ptr<DiffSingerNotes> notes;
	};

	using DiffSingerWords = std::vector<DiffSingerWord>;

	struct DiffSingerAcousticFeature {
		srt::NO<ds::ITensor> mel;
		srt::NO<ds::ITensor> f0;
	};

	struct DiffSingerAudioData {
		std::vector<uint8_t> audioData;
		int sampleRate = 0;
	};

	using DiffSingerRawData = std::vector<uint8_t>;

	DiffSingerManagedDoubleArray *getDiffSingerManagedDoubleArray(DSSP_DiffSingerManagedDoubleArray array);
	DiffSingerPhonemes *getDiffSingerPhonemes(DSSP_DiffSingerPhonemes phonemes);
	DiffSingerNotes *getDiffSingerNotes(DSSP_DiffSingerNotes notes);
	DiffSingerSpeakers *getDiffSingerSpeakers(DSSP_DiffSingerSpeakers speakers);
	DiffSingerDynamicMixedSpeakers *getDiffSingerDynamicMixedSpeakers(
		DSSP_DiffSingerDynamicMixedSpeakers speakers
	);
	DiffSingerParameters *getDiffSingerParameters(DSSP_DiffSingerParameters parameters);
	DiffSingerWords *getDiffSingerWords(DSSP_DiffSingerWords words);
	DiffSingerAcousticFeature *getDiffSingerAcousticFeature(DSSP_DiffSingerAcousticFeature feature);
	DiffSingerAudioData *getDiffSingerAudioData(DSSP_DiffSingerAudioData audioData);
	DiffSingerRawData *getDiffSingerRawData(DSSP_DiffSingerRawData rawData);

	ds::Api::Common::L1::InputWordInfo toDsinferInputWordInfo(const DiffSingerWord &word);
	std::vector<ds::Api::Common::L1::InputWordInfo> toDsinferInputWordInfos(const DiffSingerWords &words);
	ds::Api::Common::L1::InputSpeakerInfo toDsinferInputSpeakerInfo(
		const DiffSingerDynamicMixedSpeaker &speaker
	);
	std::vector<ds::Api::Common::L1::InputSpeakerInfo> toDsinferInputSpeakerInfos(
		const DiffSingerDynamicMixedSpeakers &speakers
	);
	ds::Api::Common::L1::InputParameterInfo toDsinferInputParameterInfo(const DiffSingerParameter &parameter);
	std::vector<ds::Api::Common::L1::InputParameterInfo> toDsinferInputParameterInfos(
		const DiffSingerParameters &parameters
	);
	DiffSingerParameter fromDsinferInputParameterInfo(
		const ds::Api::Common::L1::InputParameterInfo &parameter
	);
	DiffSingerParameters fromDsinferInputParameterInfos(
		const std::vector<ds::Api::Common::L1::InputParameterInfo> &parameters
	);

} // namespace dssp

#endif // DSSP_DSINFERDATA_H
