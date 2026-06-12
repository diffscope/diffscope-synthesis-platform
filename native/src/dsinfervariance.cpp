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

#include <dsinfer/Api/Inferences/Variance/1/VarianceApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Var = ds::Api::Variance::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinfervariance");
		inline constexpr char g_displayName[] = "variance";

		using Helper = DSInferInferenceHelper<
			&g_logger,
			Var::API_CLASS,
			g_displayName,
			Var::VarianceImportOptions,
			Var::VarianceRuntimeOptions,
			Var::VarianceInitArgs>;

		DSSP_DiffSingerParameters newDiffSingerParameters(
			const std::vector<Var::InputParameterInfo> &values
		) {
			auto result = std::make_unique<DiffSingerParameters>(
				fromDsinferInputParameterInfos(values)
			);
			return result.release();
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerVarianceInference DSSP_GetDiffSingerVarianceInference(DSSP_SRTSinger singer) {
	return dssp::Helper::getInference(singer);
}

const char *DSSP_GetDiffSingerVarianceInferenceSpeakerID(
	DSSP_SRTSinger singer,
	const char *singer_speaker_id
) {
	return dssp::Helper::speakerID(singer, singer_speaker_id);
}

DSSP_DiffSingerVarianceInferenceTask DSSP_CreateDiffSingerVarianceInferenceTask(
	DSSP_DiffSingerVarianceInference inference
) {
	return dssp::Helper::createTask(inference);
}

void DSSP_DeleteDiffSingerVarianceInferenceTask(DSSP_DiffSingerVarianceInferenceTask task) {
	dssp::Helper::deleteTask(task);
}

bool DSSP_IsDiffSingerVarianceInferenceTaskError(DSSP_DiffSingerVarianceInferenceTask task) {
	return dssp::Helper::isTaskError(task);
}

const char *DSSP_GetDiffSingerVarianceInferenceErrorMessage(
	DSSP_DiffSingerVarianceInferenceTask task
) {
	return dssp::Helper::taskErrorMessage(task).c_str();
}

DSSP_DiffSingerVarianceInference DSSP_GetDiffSingerVarianceInferenceTaskInference(
	DSSP_DiffSingerVarianceInferenceTask task
) {
	return dssp::Helper::taskInference(task);
}

DSSP_DiffSingerParameters DSSP_RunDiffSingerVarianceInferenceTask(
	DSSP_DiffSingerVarianceInferenceTask task,
	double duration,
	DSSP_DiffSingerWords words,
	DSSP_DiffSingerParameters parameters,
	DSSP_DiffSingerDynamicMixedSpeakers dynamicMixedSpeakers,
	int64_t steps
) {
	auto input = srt::NO<dssp::Var::VarianceStartInput>::create();
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
	auto varianceResult = taskResult.as<dssp::Var::VarianceResult>();
	auto result = dssp::newDiffSingerParameters(varianceResult->predictions);
	return result;
}

void DSSP_TerminateDiffSingerVarianceInferenceTask(DSSP_DiffSingerVarianceInferenceTask task) {
	dssp::Helper::terminateTask(task);
}
