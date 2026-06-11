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
#include "logger.h"
#include "synthrt.h"

#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>

#include <dsinfer/Api/Inferences/Acoustic/1/AcousticApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Ac = ds::Api::Acoustic::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinferacoustic");

		struct AcousticInferenceTaskError {
			std::string errorMessage;
		};

		std::mutex g_acousticInferenceTaskMutex;
		std::unordered_map<srt::InferenceSpec *, srt::NO<srt::InferenceImportOptions>>
			g_acousticInferenceImportOptions;
		std::unordered_map<srt::Inference *, srt::NO<srt::Inference>> g_acousticInferenceTasks;
		std::unordered_map<AcousticInferenceTaskError *, std::unique_ptr<AcousticInferenceTaskError>>
			g_acousticInferenceTaskErrors;

		srt::InferenceSpec *getDiffSingerAcousticInference(DSSP_DiffSingerAcousticInference inference) {
			return static_cast<srt::InferenceSpec *>(inference);
		}

		srt::Inference *getDiffSingerAcousticInferenceTask(DSSP_DiffSingerAcousticInferenceTask task) {
			return static_cast<srt::Inference *>(task);
		}

		AcousticInferenceTaskError *getDiffSingerAcousticInferenceTaskError(
			DSSP_DiffSingerAcousticInferenceTask task
		) {
			return static_cast<AcousticInferenceTaskError *>(task);
		}

		srt::SingerImport findAcousticImport(const srt::SingerSpec *singer) {
			for (const auto &import : singer->imports()) {
				if (import.inference()->className() == Ac::API_CLASS) {
					return import;
				}
			}
			return {};
		}

		DSSP_DiffSingerAcousticInferenceTask newAcousticInferenceTaskError(std::string errorMessage) {
			auto error = std::make_unique<AcousticInferenceTaskError>();
			error->errorMessage = std::move(errorMessage);
			auto *handle = error.get();

			std::lock_guard lock(g_acousticInferenceTaskMutex);
			g_acousticInferenceTaskErrors.emplace(handle, std::move(error));
			return handle;
		}

		void setAcousticInferenceImportOptions(
			srt::InferenceSpec *inference,
			srt::NO<srt::InferenceImportOptions> importOptions
		) {
			std::lock_guard lock(g_acousticInferenceTaskMutex);
			g_acousticInferenceImportOptions[inference] = std::move(importOptions);
		}

		srt::NO<srt::InferenceImportOptions> findAcousticInferenceImportOptions(
			srt::InferenceSpec *inference
		) {
			std::lock_guard lock(g_acousticInferenceTaskMutex);
			if (const auto it = g_acousticInferenceImportOptions.find(inference);
				it != g_acousticInferenceImportOptions.end()) {
				return it->second;
			}
			return {};
		}

		bool isAcousticInferenceTaskError(DSSP_DiffSingerAcousticInferenceTask task) {
			std::lock_guard lock(g_acousticInferenceTaskMutex);
			return g_acousticInferenceTaskErrors.contains(getDiffSingerAcousticInferenceTaskError(task));
		}

		const std::string &acousticInferenceTaskErrorMessage(DSSP_DiffSingerAcousticInferenceTask task) {
			static const std::string empty;

			std::lock_guard lock(g_acousticInferenceTaskMutex);
			const auto it = g_acousticInferenceTaskErrors.find(
				getDiffSingerAcousticInferenceTaskError(task)
			);
			if (it == g_acousticInferenceTaskErrors.end()) {
				return empty;
			}
			return it->second->errorMessage;
		}

		void addAcousticInferenceTask(srt::NO<srt::Inference> inference) {
			std::lock_guard lock(g_acousticInferenceTaskMutex);
			g_acousticInferenceTasks.emplace(inference.get(), std::move(inference));
		}

		srt::NO<srt::Inference> findAcousticInferenceTask(srt::Inference *task) {
			std::lock_guard lock(g_acousticInferenceTaskMutex);
			if (const auto it = g_acousticInferenceTasks.find(task);
				it != g_acousticInferenceTasks.end()) {
				return it->second;
			}
			return {};
		}

		void deleteAcousticInferenceTask(DSSP_DiffSingerAcousticInferenceTask task) {
			std::lock_guard lock(g_acousticInferenceTaskMutex);
			if (const auto it = g_acousticInferenceTaskErrors.find(
					getDiffSingerAcousticInferenceTaskError(task)
				);
				it != g_acousticInferenceTaskErrors.end()) {
				g_acousticInferenceTaskErrors.erase(it);
				return;
			}

			g_acousticInferenceTasks.erase(getDiffSingerAcousticInferenceTask(task));
		}

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
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findAcousticImport(singerSpec);
	if (import.isNull()) {
		return nullptr;
	}
	auto *inference = import.inference();
	dssp::setAcousticInferenceImportOptions(inference, import.options());
	return inference;
}

const char *DSSP_GetDiffSingerAcousticInferenceSpeakerID(
	DSSP_SRTSinger singer,
	const char *singer_speaker_id
) {
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findAcousticImport(singerSpec);
	if (import.isNull()) {
		return singer_speaker_id;
	}

	const auto options = import.options().as<dssp::Ac::AcousticImportOptions>();
	if (!options) {
		return singer_speaker_id;
	}
	const auto it = options->speakerMapping.find(singer_speaker_id);
	return it == options->speakerMapping.end() ? singer_speaker_id : it->second.c_str();
}

DSSP_DiffSingerAcousticInferenceTask DSSP_CreateDiffSingerAcousticInferenceTask(
	DSSP_DiffSingerAcousticInference inference
) {
	auto *spec = dssp::getDiffSingerAcousticInference(inference);
	if (spec == nullptr) {
		return dssp::newAcousticInferenceTaskError("acoustic inference is nullptr");
	}

	srt::NO<srt::Inference> task;
	auto importOptions = dssp::findAcousticInferenceImportOptions(spec);
	if (!importOptions) {
		importOptions = srt::NO<dssp::Ac::AcousticImportOptions>::create();
	}
	if (auto exp = spec->createInference(
			importOptions,
			srt::NO<dssp::Ac::AcousticRuntimeOptions>::create()
		);
		!exp) {
		return dssp::newAcousticInferenceTaskError(exp.error().message());
	} else {
		task = exp.take();
	}

	if (auto exp = task->initialize(srt::NO<dssp::Ac::AcousticInitArgs>::create()); !exp) {
		return dssp::newAcousticInferenceTaskError(exp.error().message());
	}

	auto *handle = task.get();
	dssp::addAcousticInferenceTask(std::move(task));
	dssp::g_logger.info("DiffSinger acoustic inference task created");
	return handle;
}

void DSSP_DeleteDiffSingerAcousticInferenceTask(DSSP_DiffSingerAcousticInferenceTask task) {
	dssp::deleteAcousticInferenceTask(task);
}

bool DSSP_IsDiffSingerAcousticInferenceTaskError(DSSP_DiffSingerAcousticInferenceTask task) {
	return dssp::isAcousticInferenceTaskError(task);
}

const char *DSSP_GetDiffSingerAcousticInferenceErrorMessage(
	DSSP_DiffSingerAcousticInferenceTask task
) {
	return dssp::acousticInferenceTaskErrorMessage(task).c_str();
}

DSSP_DiffSingerAcousticInference DSSP_GetDiffSingerAcousticInferenceTaskInference(
	DSSP_DiffSingerAcousticInferenceTask task
) {
	if (dssp::isAcousticInferenceTaskError(task)) {
		return nullptr;
	}
	const auto *inference = dssp::getDiffSingerAcousticInferenceTask(task);
	return const_cast<srt::InferenceSpec *>(inference->spec());
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
	auto *inference = dssp::getDiffSingerAcousticInferenceTask(task);
	dssp::g_logger.info("DiffSinger acoustic inference task started");

	auto input = srt::NO<dssp::Ac::AcousticStartInput>::create();
	input->duration = duration;
	input->words = dssp::toDsinferInputWordInfos(*dssp::getDiffSingerWords(words));
	input->parameters = dssp::toDsinferInputParameterInfos(*dssp::getDiffSingerParameters(parameters));
	input->speakers = dssp::toDsinferInputSpeakerInfos(
		*dssp::getDiffSingerDynamicMixedSpeakers(dynamicMixedSpeakers)
	);
	input->depth = depth;
	input->steps = steps;

	srt::NO<srt::TaskResult> taskResult;
	if (auto exp = inference->start(input); !exp) {
		dssp::g_logger.error(
			std::string("Failed to run DiffSinger acoustic inference task: ") + exp.error().message()
		);
		return nullptr;
	} else {
		taskResult = exp.take();
	}

	auto acousticResult = taskResult.as<dssp::Ac::AcousticResult>();
	if (inference->state() == srt::ITask::Failed) {
		dssp::g_logger.error("Failed to run DiffSinger acoustic inference task");
		return nullptr;
	}
	if (!acousticResult->mel || !acousticResult->f0) {
		dssp::g_logger.error("Failed to run DiffSinger acoustic inference task: result is empty");
		return nullptr;
	}
	if (acousticResult->mel->elementCount() == 0 || acousticResult->f0->elementCount() == 0) {
		dssp::g_logger.error("Failed to run DiffSinger acoustic inference task: result is empty");
		return nullptr;
	}

	auto result = dssp::newDiffSingerAcousticFeature(acousticResult->mel, acousticResult->f0);
	dssp::g_logger.info("DiffSinger acoustic inference task completed");
	return result;
}

void DSSP_TerminateDiffSingerAcousticInferenceTask(DSSP_DiffSingerAcousticInferenceTask task) {
	if (dssp::isAcousticInferenceTaskError(task)) {
		return;
	}

	auto inference = dssp::findAcousticInferenceTask(dssp::getDiffSingerAcousticInferenceTask(task));
	if (!inference) {
		return;
	}
	inference->stop();
	dssp::g_logger.info("DiffSinger acoustic inference task terminated manually");
}
