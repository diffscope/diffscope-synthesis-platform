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

#include <dsinfer/Api/Inferences/Vocoder/1/VocoderApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Voc = ds::Api::Vocoder::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinfervocoder");
		inline constexpr char g_displayName[] = "vocoder";

		using Helper = DSInferInferenceHelper<
			&g_logger,
			Voc::API_CLASS,
			g_displayName,
			Voc::VocoderImportOptions,
			Voc::VocoderRuntimeOptions,
			Voc::VocoderInitArgs>;

		DSSP_DiffSingerAudioData newDiffSingerAudioData(
			std::vector<uint8_t> audioData,
			int sampleRate
		) {
			auto result = std::make_unique<DiffSingerAudioData>();
			result->audioData = std::move(audioData);
			result->sampleRate = sampleRate;
			return result.release();
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerVocoderInference DSSP_GetDiffSingerVocoderInference(DSSP_SRTSinger singer) {
	return dssp::Helper::getInference(singer);
}

DSSP_DiffSingerVocoderInferenceTask DSSP_CreateDiffSingerVocoderInferenceTask(
	DSSP_DiffSingerVocoderInference inference
) {
	return dssp::Helper::createTask(inference);
}

void DSSP_DeleteDiffSingerVocoderInferenceTask(DSSP_DiffSingerVocoderInferenceTask task) {
	dssp::Helper::deleteTask(task);
}

bool DSSP_IsDiffSingerVocoderInferenceTaskError(DSSP_DiffSingerVocoderInferenceTask task) {
	return dssp::Helper::isTaskError(task);
}

const char *DSSP_GetDiffSingerVocoderInferenceErrorMessage(
	DSSP_DiffSingerVocoderInferenceTask task
) {
	return dssp::Helper::taskErrorMessage(task).c_str();
}

DSSP_DiffSingerVocoderInference DSSP_GetDiffSingerVocoderInferenceTaskInference(
	DSSP_DiffSingerVocoderInferenceTask task
) {
	return dssp::Helper::taskInference(task);
}

DSSP_DiffSingerAudioData DSSP_RunDiffSingerVocoderInferenceTask(
	DSSP_DiffSingerVocoderInferenceTask task,
	DSSP_DiffSingerAcousticFeature feature
) {
	auto *inference = dssp::Helper::task(task);

	auto *acousticFeature = dssp::getDiffSingerAcousticFeature(feature);
	auto configuration = inference->spec()->configuration().as<dssp::Voc::VocoderConfiguration>();
	
	auto input = srt::NO<dssp::Voc::VocoderStartInput>::create();
	input->mel = acousticFeature->mel;
	input->f0 = acousticFeature->f0;

	auto taskResult = dssp::Helper::runTask(task, input);
	if (!taskResult) {
		return nullptr;
	}
	auto vocoderResult = taskResult.as<dssp::Voc::VocoderResult>();
	auto result = dssp::newDiffSingerAudioData(
		std::move(vocoderResult->audioData),
		configuration->sampleRate
	);
	return result;
}

void DSSP_TerminateDiffSingerVocoderInferenceTask(DSSP_DiffSingerVocoderInferenceTask task) {
	dssp::Helper::terminateTask(task);
}
