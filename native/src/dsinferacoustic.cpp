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

#include <dsinfer/Api/Inferences/Acoustic/1/AcousticApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Ac = ds::Api::Acoustic::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinferacoustic");
		inline constexpr char g_displayName[] = "acoustic";

		using Helper = DSInferInferenceHelper<
			&g_logger,
			Ac::API_CLASS,
			g_displayName,
			Ac::AcousticImportOptions,
			Ac::AcousticRuntimeOptions,
			Ac::AcousticInitArgs>;

		DSSP_DiffSingerAcousticFeature newDiffSingerAcousticFeature(
			srt::NO<ds::ITensor> mel,
			srt::NO<ds::ITensor> f0
		) {
			auto result = std::make_unique<DiffSingerAcousticFeature>();
			result->mel = std::move(mel);
			result->f0 = std::move(f0);
			return result.release();
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerAcousticInference DSSP_GetDiffSingerAcousticInference(DSSP_SRTSinger singer) {
	return dssp::Helper::getInference(singer);
}

const char *DSSP_GetDiffSingerAcousticInferenceSpeakerID(
	DSSP_SRTSinger singer,
	const char *singer_speaker_id
) {
	return dssp::Helper::speakerID(singer, singer_speaker_id);
}

DSSP_DiffSingerAcousticInferenceTask DSSP_CreateDiffSingerAcousticInferenceTask(
	DSSP_DiffSingerAcousticInference inference
) {
	return dssp::Helper::createTask(inference);
}

void DSSP_DeleteDiffSingerAcousticInferenceTask(DSSP_DiffSingerAcousticInferenceTask task) {
	dssp::Helper::deleteTask(task);
}

bool DSSP_IsDiffSingerAcousticInferenceTaskError(DSSP_DiffSingerAcousticInferenceTask task) {
	return dssp::Helper::isTaskError(task);
}

const char *DSSP_GetDiffSingerAcousticInferenceErrorMessage(
	DSSP_DiffSingerAcousticInferenceTask task
) {
	return dssp::Helper::taskErrorMessage(task).c_str();
}

DSSP_DiffSingerAcousticInference DSSP_GetDiffSingerAcousticInferenceTaskInference(
	DSSP_DiffSingerAcousticInferenceTask task
) {
	return dssp::Helper::taskInference(task);
}

DSSP_DiffSingerAcousticFeature DSSP_RunDiffSingerAcousticInferenceTask(
	DSSP_DiffSingerAcousticInferenceTask task,
	double duration,
	DSSP_DiffSingerWords words,
	DSSP_DiffSingerParameters parameters,
	DSSP_DiffSingerDynamicMixedSpeakers dynamicMixedSpeakers,
	float depth,
	int64_t steps
) {
	auto input = srt::NO<dssp::Ac::AcousticStartInput>::create();
	input->duration = duration;
	input->words = dssp::toDsinferInputWordInfos(*dssp::getDiffSingerWords(words));
	input->parameters = dssp::toDsinferInputParameterInfos(*dssp::getDiffSingerParameters(parameters));
	input->speakers = dssp::toDsinferInputSpeakerInfos(
		*dssp::getDiffSingerDynamicMixedSpeakers(dynamicMixedSpeakers)
	);
	input->depth = depth;
	input->steps = steps;

	auto taskResult = dssp::Helper::runTask(task, input);
	if (!taskResult) {
		return nullptr;
	}
	auto acousticResult = taskResult.as<dssp::Ac::AcousticResult>();
	auto result = dssp::newDiffSingerAcousticFeature(acousticResult->mel, acousticResult->f0);
	return result;
}

void DSSP_TerminateDiffSingerAcousticInferenceTask(DSSP_DiffSingerAcousticInferenceTask task) {
	dssp::Helper::terminateTask(task);
}
