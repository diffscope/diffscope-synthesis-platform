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
#include "dsinferinferencehelper.h"
#include "logger.h"

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include <dsinfer/Api/Inferences/Pitch/1/PitchApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Pit = ds::Api::Pitch::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinferpitch");
		inline constexpr char g_displayName[] = "pitch";

		using Helper = DSInferInferenceHelper<
			&g_logger,
			Pit::API_CLASS,
			g_displayName,
			Pit::PitchImportOptions,
			Pit::PitchRuntimeOptions,
			Pit::PitchInitArgs>;

		DSSP_DiffSingerParameters newDiffSingerPitchParameters(
			std::vector<double> values,
			double interval
		) {
			auto result = std::make_unique<DiffSingerParameters>(1);
			auto &parameter = result->at(0);
			parameter.tag = DSSP_DiffSingerParameterTag_Pitch;
			parameter.values = std::make_unique<DiffSingerManagedDoubleArray>(std::move(values));
			parameter.interval = interval;
			return result.release();
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerPitchInference DSSP_GetDiffSingerPitchInference(DSSP_SRTSinger singer) {
	return dssp::Helper::getInference(singer);
}

const char *DSSP_GetDiffSingerPitchInferenceSpeakerID(DSSP_SRTSinger singer, const char *singer_speaker_id) {
	return dssp::Helper::speakerID(singer, singer_speaker_id);
}

DSSP_DiffSingerPitchInferenceTask DSSP_CreateDiffSingerPitchInferenceTask(
	DSSP_DiffSingerPitchInference inference
) {
	return dssp::Helper::createTask(inference);
}

void DSSP_DeleteDiffSingerPitchInferenceTask(DSSP_DiffSingerPitchInferenceTask task) {
	dssp::Helper::deleteTask(task);
}

bool DSSP_IsDiffSingerPitchInferenceTaskError(DSSP_DiffSingerPitchInferenceTask task) {
	return dssp::Helper::isTaskError(task);
}

const char *DSSP_GetDiffSingerPitchInferenceErrorMessage(DSSP_DiffSingerPitchInferenceTask task) {
	return dssp::Helper::taskErrorMessage(task).c_str();
}

DSSP_DiffSingerPitchInference DSSP_GetDiffSingerPitchInferenceTaskInference(
	DSSP_DiffSingerPitchInferenceTask task
) {
	return dssp::Helper::taskInference(task);
}

DSSP_DiffSingerParameters DSSP_RunDiffSingerPitchInferenceTask(
	DSSP_DiffSingerPitchInferenceTask task,
	double duration,
	DSSP_DiffSingerWords words,
	DSSP_DiffSingerParameters parameters,
	DSSP_DiffSingerDynamicMixedSpeakers dynamicMixedSpeakers,
	int64_t steps
) {
	auto input = srt::NO<dssp::Pit::PitchStartInput>::create();
	input->duration = duration;
	input->words = dssp::toDsinferInputWordInfos(*dssp::getDiffSingerWords(words));
	input->parameters = dssp::toDsinferInputParameterInfos(*dssp::getDiffSingerParameters(parameters));
	input->speakers = dssp::toDsinferInputSpeakerInfos(
		*dssp::getDiffSingerDynamicMixedSpeakers(dynamicMixedSpeakers)
	);
	input->steps = steps;

	auto taskResult = dssp::Helper::runTask(task, input);
	if (!taskResult) {
		return nullptr;
	}
	auto pitchResult = taskResult.as<dssp::Pit::PitchResult>();
	auto result = dssp::newDiffSingerPitchParameters(
		std::move(pitchResult->pitch),
		pitchResult->interval
	);
	return result;
}

void DSSP_TerminateDiffSingerPitchInferenceTask(DSSP_DiffSingerPitchInferenceTask task) {
	dssp::Helper::terminateTask(task);
}
