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

#include <dsinfer/Api/Inferences/Variance/1/VarianceApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Var = ds::Api::Variance::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinfervariance");

		struct VarianceInferenceTaskError {
			std::string errorMessage;
		};

		std::mutex g_varianceInferenceTaskMutex;
		std::unordered_map<srt::InferenceSpec *, srt::NO<srt::InferenceImportOptions>>
			g_varianceInferenceImportOptions;
		std::unordered_map<srt::Inference *, srt::NO<srt::Inference>> g_varianceInferenceTasks;
		std::unordered_map<VarianceInferenceTaskError *, std::unique_ptr<VarianceInferenceTaskError>>
			g_varianceInferenceTaskErrors;

		srt::InferenceSpec *getDiffSingerVarianceInference(DSSP_DiffSingerVarianceInference inference) {
			return static_cast<srt::InferenceSpec *>(inference);
		}

		srt::Inference *getDiffSingerVarianceInferenceTask(DSSP_DiffSingerVarianceInferenceTask task) {
			return static_cast<srt::Inference *>(task);
		}

		VarianceInferenceTaskError *getDiffSingerVarianceInferenceTaskError(
			DSSP_DiffSingerVarianceInferenceTask task
		) {
			return static_cast<VarianceInferenceTaskError *>(task);
		}

		srt::SingerImport findVarianceImport(const srt::SingerSpec *singer) {
			for (const auto &import : singer->imports()) {
				if (import.inference()->className() == Var::API_CLASS) {
					return import;
				}
			}
			return {};
		}

		DSSP_DiffSingerVarianceInferenceTask newVarianceInferenceTaskError(std::string errorMessage) {
			auto error = std::make_unique<VarianceInferenceTaskError>();
			error->errorMessage = std::move(errorMessage);
			auto *handle = error.get();

			std::lock_guard lock(g_varianceInferenceTaskMutex);
			g_varianceInferenceTaskErrors.emplace(handle, std::move(error));
			return handle;
		}

		void setVarianceInferenceImportOptions(
			srt::InferenceSpec *inference,
			srt::NO<srt::InferenceImportOptions> importOptions
		) {
			std::lock_guard lock(g_varianceInferenceTaskMutex);
			g_varianceInferenceImportOptions[inference] = std::move(importOptions);
		}

		srt::NO<srt::InferenceImportOptions> findVarianceInferenceImportOptions(
			srt::InferenceSpec *inference
		) {
			std::lock_guard lock(g_varianceInferenceTaskMutex);
			if (const auto it = g_varianceInferenceImportOptions.find(inference);
				it != g_varianceInferenceImportOptions.end()) {
				return it->second;
			}
			return {};
		}

		bool isVarianceInferenceTaskError(DSSP_DiffSingerVarianceInferenceTask task) {
			std::lock_guard lock(g_varianceInferenceTaskMutex);
			return g_varianceInferenceTaskErrors.contains(getDiffSingerVarianceInferenceTaskError(task));
		}

		const std::string &varianceInferenceTaskErrorMessage(DSSP_DiffSingerVarianceInferenceTask task) {
			static const std::string empty;

			std::lock_guard lock(g_varianceInferenceTaskMutex);
			const auto it = g_varianceInferenceTaskErrors.find(
				getDiffSingerVarianceInferenceTaskError(task)
			);
			if (it == g_varianceInferenceTaskErrors.end()) {
				return empty;
			}
			return it->second->errorMessage;
		}

		void addVarianceInferenceTask(srt::NO<srt::Inference> inference) {
			std::lock_guard lock(g_varianceInferenceTaskMutex);
			g_varianceInferenceTasks.emplace(inference.get(), std::move(inference));
		}

		srt::NO<srt::Inference> findVarianceInferenceTask(srt::Inference *task) {
			std::lock_guard lock(g_varianceInferenceTaskMutex);
			if (const auto it = g_varianceInferenceTasks.find(task);
				it != g_varianceInferenceTasks.end()) {
				return it->second;
			}
			return {};
		}

		void deleteVarianceInferenceTask(DSSP_DiffSingerVarianceInferenceTask task) {
			std::lock_guard lock(g_varianceInferenceTaskMutex);
			if (const auto it = g_varianceInferenceTaskErrors.find(
					getDiffSingerVarianceInferenceTaskError(task)
				);
				it != g_varianceInferenceTaskErrors.end()) {
				g_varianceInferenceTaskErrors.erase(it);
				return;
			}

			g_varianceInferenceTasks.erase(getDiffSingerVarianceInferenceTask(task));
		}

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
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findVarianceImport(singerSpec);
	if (import.isNull()) {
		return nullptr;
	}
	auto *inference = import.inference();
	dssp::setVarianceInferenceImportOptions(inference, import.options());
	return inference;
}

const char *DSSP_GetDiffSingerVarianceInferenceSpeakerID(
	DSSP_SRTSinger singer,
	const char *singer_speaker_id
) {
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findVarianceImport(singerSpec);
	if (import.isNull()) {
		return singer_speaker_id;
	}

	const auto options = import.options().as<dssp::Var::VarianceImportOptions>();
	if (!options) {
		return singer_speaker_id;
	}
	const auto it = options->speakerMapping.find(singer_speaker_id);
	return it == options->speakerMapping.end() ? singer_speaker_id : it->second.c_str();
}

DSSP_DiffSingerVarianceInferenceTask DSSP_CreateDiffSingerVarianceInferenceTask(
	DSSP_DiffSingerVarianceInference inference
) {
	auto *spec = dssp::getDiffSingerVarianceInference(inference);
	if (spec == nullptr) {
		return dssp::newVarianceInferenceTaskError("variance inference is nullptr");
	}

	srt::NO<srt::Inference> task;
	auto importOptions = dssp::findVarianceInferenceImportOptions(spec);
	if (!importOptions) {
		importOptions = srt::NO<dssp::Var::VarianceImportOptions>::create();
	}
	if (auto exp = spec->createInference(
			importOptions,
			srt::NO<dssp::Var::VarianceRuntimeOptions>::create()
		);
		!exp) {
		return dssp::newVarianceInferenceTaskError(exp.error().message());
	} else {
		task = exp.take();
	}

	if (auto exp = task->initialize(srt::NO<dssp::Var::VarianceInitArgs>::create()); !exp) {
		return dssp::newVarianceInferenceTaskError(exp.error().message());
	}

	auto *handle = task.get();
	dssp::addVarianceInferenceTask(std::move(task));
	dssp::g_logger.info("DiffSinger variance inference task created");
	return handle;
}

void DSSP_DeleteDiffSingerVarianceInferenceTask(DSSP_DiffSingerVarianceInferenceTask task) {
	dssp::deleteVarianceInferenceTask(task);
}

bool DSSP_IsDiffSingerVarianceInferenceTaskError(DSSP_DiffSingerVarianceInferenceTask task) {
	return dssp::isVarianceInferenceTaskError(task);
}

const char *DSSP_GetDiffSingerVarianceInferenceErrorMessage(
	DSSP_DiffSingerVarianceInferenceTask task
) {
	return dssp::varianceInferenceTaskErrorMessage(task).c_str();
}

DSSP_DiffSingerVarianceInference DSSP_GetDiffSingerVarianceInferenceTaskInference(
	DSSP_DiffSingerVarianceInferenceTask task
) {
	if (dssp::isVarianceInferenceTaskError(task)) {
		return nullptr;
	}
	const auto *inference = dssp::getDiffSingerVarianceInferenceTask(task);
	return const_cast<srt::InferenceSpec *>(inference->spec());
}

DSSP_DiffSingerParameters DSSP_RunDiffSingerVarianceInferenceTask(
	DSSP_DiffSingerVarianceInferenceTask task,
	double duration,
	DSSP_DiffSingerWords words,
	DSSP_DiffSingerParameters parameters,
	DSSP_DiffSingerDynamicMixedSpeakers dynamicMixedSpeakers,
	int64_t steps
) {
	auto *inference = dssp::getDiffSingerVarianceInferenceTask(task);
	dssp::g_logger.info("DiffSinger variance inference task started");

	auto input = srt::NO<dssp::Var::VarianceStartInput>::create();
	input->duration = duration;
	input->words = dssp::toDsinferInputWordInfos(*dssp::getDiffSingerWords(words));
	input->parameters = dssp::toDsinferInputParameterInfos(*dssp::getDiffSingerParameters(parameters));
	input->speakers = dssp::toDsinferInputSpeakerInfos(
		*dssp::getDiffSingerDynamicMixedSpeakers(dynamicMixedSpeakers)
	);
	input->steps = steps;

	srt::NO<srt::TaskResult> taskResult;
	if (auto exp = inference->start(input); !exp) {
		dssp::g_logger.error(
			std::string("Failed to run DiffSinger variance inference task: ") + exp.error().message()
		);
		return nullptr;
	} else {
		taskResult = exp.take();
	}

	auto varianceResult = taskResult.as<dssp::Var::VarianceResult>();
	if (inference->state() == srt::ITask::Failed) {
		dssp::g_logger.error(
			std::string("Failed to run DiffSinger variance inference task: ") + varianceResult->error.message()
		);
		return nullptr;
	}
	if (varianceResult->predictions.empty()) {
		dssp::g_logger.error("Failed to run DiffSinger variance inference task: result is empty");
		return nullptr;
	}

	auto result = dssp::newDiffSingerParameters(varianceResult->predictions);
	dssp::g_logger.info("DiffSinger variance inference task completed");
	return result;
}

void DSSP_TerminateDiffSingerVarianceInferenceTask(DSSP_DiffSingerVarianceInferenceTask task) {
	if (dssp::isVarianceInferenceTaskError(task)) {
		return;
	}

	auto inference = dssp::findVarianceInferenceTask(dssp::getDiffSingerVarianceInferenceTask(task));
	if (!inference) {
		return;
	}
	inference->stop();
	dssp::g_logger.info("DiffSinger variance inference task terminated manually");
}
