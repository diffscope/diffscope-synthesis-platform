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

#include <exception>
#include <memory>
#include <string>
#include <utility>

#include <dsinfer/Api/Inferences/Duration/1/DurationApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Dur = ds::Api::Duration::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinferduration");
		inline constexpr char g_displayName[] = "duration";

		using Helper = DSInferInferenceHelper<
			&g_logger,
			Dur::API_CLASS,
			g_displayName,
			Dur::DurationImportOptions,
			Dur::DurationRuntimeOptions,
			Dur::DurationInitArgs>;

		DSSP_DiffSingerManagedDoubleArray newDiffSingerManagedDoubleArray(std::vector<double> values) {
			auto result = std::make_unique<DiffSingerManagedDoubleArray>(std::move(values));
			return result.release();
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerDurationInference DSSP_GetDiffSingerDurationInference(DSSP_SRTSinger singer) {
	return dssp::Helper::getInference(singer);
}

const char *DSSP_GetDiffSingerDurationInferenceSpeakerID(DSSP_SRTSinger singer, const char *singer_speaker_id) {
	return dssp::Helper::speakerID(singer, singer_speaker_id);
}

DSSP_DiffSingerDurationInferenceTask DSSP_CreateDiffSingerDurationInferenceTask(
	DSSP_DiffSingerDurationInference inference
) {
	return dssp::Helper::createTask(inference);
}

void DSSP_DeleteDiffSingerDurationInferenceTask(DSSP_DiffSingerDurationInferenceTask task) {
	dssp::Helper::deleteTask(task);
}

bool DSSP_IsDiffSingerDurationInferenceTaskError(DSSP_DiffSingerDurationInferenceTask task) {
	return dssp::Helper::isTaskError(task);
}

const char *DSSP_GetDiffSingerDurationInferenceErrorMessage(DSSP_DiffSingerDurationInferenceTask task) {
	return dssp::Helper::taskErrorMessage(task).c_str();
}

DSSP_DiffSingerDurationInference DSSP_GetDiffSingerDurationInferenceTaskInference(
	DSSP_DiffSingerDurationInferenceTask task
) {
	return dssp::Helper::taskInference(task);
}

DSSP_DiffSingerManagedDoubleArray DSSP_RunDiffSingerDurationInferenceTask(
	DSSP_DiffSingerDurationInferenceTask task,
	double duration,
	DSSP_DiffSingerWords words
) {
	auto input = srt::NO<dssp::Dur::DurationStartInput>::create();
	input->duration = duration;
	input->words = dssp::toDsinferInputWordInfos(*dssp::getDiffSingerWords(words));

	auto taskResult = dssp::Helper::runTask(task, input);
	if (!taskResult) {
		return nullptr;
	}
	auto durationResult = taskResult.as<dssp::Dur::DurationResult>();
	auto result = dssp::newDiffSingerManagedDoubleArray(std::move(durationResult->durations));
	return result;
}

void DSSP_TerminateDiffSingerDurationInferenceTask(DSSP_DiffSingerDurationInferenceTask task) {
	dssp::Helper::terminateTask(task);
}
