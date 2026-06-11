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
#include <vector>

#include <dsinfer/Api/Inferences/Vocoder/1/VocoderApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Voc = ds::Api::Vocoder::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinfervocoder");

		struct VocoderInferenceTaskError {
			std::string errorMessage;
		};

		std::mutex g_vocoderInferenceTaskMutex;
		std::unordered_map<srt::InferenceSpec *, srt::NO<srt::InferenceImportOptions>>
			g_vocoderInferenceImportOptions;
		std::unordered_map<srt::Inference *, srt::NO<srt::Inference>> g_vocoderInferenceTasks;
		std::unordered_map<VocoderInferenceTaskError *, std::unique_ptr<VocoderInferenceTaskError>>
			g_vocoderInferenceTaskErrors;

		srt::InferenceSpec *getDiffSingerVocoderInference(DSSP_DiffSingerVocoderInference inference) {
			return static_cast<srt::InferenceSpec *>(inference);
		}

		srt::Inference *getDiffSingerVocoderInferenceTask(DSSP_DiffSingerVocoderInferenceTask task) {
			return static_cast<srt::Inference *>(task);
		}

		VocoderInferenceTaskError *getDiffSingerVocoderInferenceTaskError(
			DSSP_DiffSingerVocoderInferenceTask task
		) {
			return static_cast<VocoderInferenceTaskError *>(task);
		}

		srt::SingerImport findVocoderImport(const srt::SingerSpec *singer) {
			for (const auto &import : singer->imports()) {
				if (import.inference()->className() == Voc::API_CLASS) {
					return import;
				}
			}
			return {};
		}

		DSSP_DiffSingerVocoderInferenceTask newVocoderInferenceTaskError(std::string errorMessage) {
			auto error = std::make_unique<VocoderInferenceTaskError>();
			error->errorMessage = std::move(errorMessage);
			auto *handle = error.get();

			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			g_vocoderInferenceTaskErrors.emplace(handle, std::move(error));
			return handle;
		}

		void setVocoderInferenceImportOptions(
			srt::InferenceSpec *inference,
			srt::NO<srt::InferenceImportOptions> importOptions
		) {
			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			g_vocoderInferenceImportOptions[inference] = std::move(importOptions);
		}

		srt::NO<srt::InferenceImportOptions> findVocoderInferenceImportOptions(
			srt::InferenceSpec *inference
		) {
			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			if (const auto it = g_vocoderInferenceImportOptions.find(inference);
				it != g_vocoderInferenceImportOptions.end()) {
				return it->second;
			}
			return {};
		}

		bool isVocoderInferenceTaskError(DSSP_DiffSingerVocoderInferenceTask task) {
			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			return g_vocoderInferenceTaskErrors.contains(getDiffSingerVocoderInferenceTaskError(task));
		}

		const std::string &vocoderInferenceTaskErrorMessage(DSSP_DiffSingerVocoderInferenceTask task) {
			static const std::string empty;

			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			const auto it = g_vocoderInferenceTaskErrors.find(
				getDiffSingerVocoderInferenceTaskError(task)
			);
			if (it == g_vocoderInferenceTaskErrors.end()) {
				return empty;
			}
			return it->second->errorMessage;
		}

		void addVocoderInferenceTask(srt::NO<srt::Inference> inference) {
			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			g_vocoderInferenceTasks.emplace(inference.get(), std::move(inference));
		}

		srt::NO<srt::Inference> findVocoderInferenceTask(srt::Inference *task) {
			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			if (const auto it = g_vocoderInferenceTasks.find(task);
				it != g_vocoderInferenceTasks.end()) {
				return it->second;
			}
			return {};
		}

		void deleteVocoderInferenceTask(DSSP_DiffSingerVocoderInferenceTask task) {
			std::lock_guard lock(g_vocoderInferenceTaskMutex);
			if (const auto it = g_vocoderInferenceTaskErrors.find(
					getDiffSingerVocoderInferenceTaskError(task)
				);
				it != g_vocoderInferenceTaskErrors.end()) {
				g_vocoderInferenceTaskErrors.erase(it);
				return;
			}

			g_vocoderInferenceTasks.erase(getDiffSingerVocoderInferenceTask(task));
		}

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
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findVocoderImport(singerSpec);
	if (import.isNull()) {
		return nullptr;
	}
	auto *inference = import.inference();
	dssp::setVocoderInferenceImportOptions(inference, import.options());
	return inference;
}

DSSP_DiffSingerVocoderInferenceTask DSSP_CreateDiffSingerVocoderInferenceTask(
	DSSP_DiffSingerVocoderInference inference
) {
	auto *spec = dssp::getDiffSingerVocoderInference(inference);
	if (spec == nullptr) {
		return dssp::newVocoderInferenceTaskError("vocoder inference is nullptr");
	}

	srt::NO<srt::Inference> task;
	auto importOptions = dssp::findVocoderInferenceImportOptions(spec);
	if (!importOptions) {
		importOptions = srt::NO<dssp::Voc::VocoderImportOptions>::create();
	}
	if (auto exp = spec->createInference(
			importOptions,
			srt::NO<dssp::Voc::VocoderRuntimeOptions>::create()
		);
		!exp) {
		return dssp::newVocoderInferenceTaskError(exp.error().message());
	} else {
		task = exp.take();
	}

	if (auto exp = task->initialize(srt::NO<dssp::Voc::VocoderInitArgs>::create()); !exp) {
		return dssp::newVocoderInferenceTaskError(exp.error().message());
	}

	auto *handle = task.get();
	dssp::addVocoderInferenceTask(std::move(task));
	dssp::g_logger.info("DiffSinger vocoder inference task created");
	return handle;
}

void DSSP_DeleteDiffSingerVocoderInferenceTask(DSSP_DiffSingerVocoderInferenceTask task) {
	dssp::deleteVocoderInferenceTask(task);
}

bool DSSP_IsDiffSingerVocoderInferenceTaskError(DSSP_DiffSingerVocoderInferenceTask task) {
	return dssp::isVocoderInferenceTaskError(task);
}

const char *DSSP_GetDiffSingerVocoderInferenceErrorMessage(
	DSSP_DiffSingerVocoderInferenceTask task
) {
	return dssp::vocoderInferenceTaskErrorMessage(task).c_str();
}

DSSP_DiffSingerVocoderInference DSSP_GetDiffSingerVocoderInferenceTaskInference(
	DSSP_DiffSingerVocoderInferenceTask task
) {
	if (dssp::isVocoderInferenceTaskError(task)) {
		return nullptr;
	}
	const auto *inference = dssp::getDiffSingerVocoderInferenceTask(task);
	return const_cast<srt::InferenceSpec *>(inference->spec());
}

DSSP_DiffSingerAudioData DSSP_RunDiffSingerVocoderInferenceTask(
	DSSP_DiffSingerVocoderInferenceTask task,
	DSSP_DiffSingerAcousticFeature feature
) {
	auto *inference = dssp::getDiffSingerVocoderInferenceTask(task);
	dssp::g_logger.info("DiffSinger vocoder inference task started");

	auto *acousticFeature = dssp::getDiffSingerAcousticFeature(feature);
	if (acousticFeature == nullptr) {
		dssp::g_logger.error("Failed to run DiffSinger vocoder inference task: feature is nullptr");
		return nullptr;
	}
	if (!acousticFeature->mel || !acousticFeature->f0) {
		dssp::g_logger.error("Failed to run DiffSinger vocoder inference task: feature is empty");
		return nullptr;
	}

	auto configuration = inference->spec()->configuration().as<dssp::Voc::VocoderConfiguration>();
	if (!configuration) {
		dssp::g_logger.error("Failed to run DiffSinger vocoder inference task: configuration is unavailable");
		return nullptr;
	}
	if (configuration->sampleRate <= 0) {
		dssp::g_logger.error("Failed to run DiffSinger vocoder inference task: sample rate is invalid");
		return nullptr;
	}

	auto input = srt::NO<dssp::Voc::VocoderStartInput>::create();
	input->mel = acousticFeature->mel;
	input->f0 = acousticFeature->f0;

	srt::NO<srt::TaskResult> taskResult;
	if (auto exp = inference->start(input); !exp) {
		dssp::g_logger.error(
			std::string("Failed to run DiffSinger vocoder inference task: ") + exp.error().message()
		);
		return nullptr;
	} else {
		taskResult = exp.take();
	}

	auto vocoderResult = taskResult.as<dssp::Voc::VocoderResult>();
	if (inference->state() == srt::ITask::Failed) {
		dssp::g_logger.error("Failed to run DiffSinger vocoder inference task");
		return nullptr;
	}
	if (vocoderResult->audioData.empty()) {
		dssp::g_logger.error("Failed to run DiffSinger vocoder inference task: result is empty");
		return nullptr;
	}

	auto result = dssp::newDiffSingerAudioData(
		std::move(vocoderResult->audioData),
		configuration->sampleRate
	);
	dssp::g_logger.info("DiffSinger vocoder inference task completed");
	return result;
}

void DSSP_TerminateDiffSingerVocoderInferenceTask(DSSP_DiffSingerVocoderInferenceTask task) {
	if (dssp::isVocoderInferenceTaskError(task)) {
		return;
	}

	auto inference = dssp::findVocoderInferenceTask(dssp::getDiffSingerVocoderInferenceTask(task));
	if (!inference) {
		return;
	}
	inference->stop();
	dssp::g_logger.info("DiffSinger vocoder inference task terminated manually");
}
