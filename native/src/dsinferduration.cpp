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

#include <exception>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>

#include <dsinfer/Api/Inferences/Duration/1/DurationApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Dur = ds::Api::Duration::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinferduration");

		struct DurationInferenceTaskError {
			std::string errorMessage;
		};

		std::mutex g_durationInferenceTaskMutex;
		std::unordered_map<srt::InferenceSpec *, srt::NO<srt::InferenceImportOptions>>
			g_durationInferenceImportOptions;
		std::unordered_map<srt::Inference *, srt::NO<srt::Inference>> g_durationInferenceTasks;
		std::unordered_map<DurationInferenceTaskError *, std::unique_ptr<DurationInferenceTaskError>>
			g_durationInferenceTaskErrors;

		srt::InferenceSpec *getDiffSingerDurationInference(DSSP_DiffSingerDurationInference inference) {
			return static_cast<srt::InferenceSpec *>(inference);
		}

		srt::Inference *getDiffSingerDurationInferenceTask(DSSP_DiffSingerDurationInferenceTask task) {
			return static_cast<srt::Inference *>(task);
		}

		DurationInferenceTaskError *getDiffSingerDurationInferenceTaskError(
			DSSP_DiffSingerDurationInferenceTask task
		) {
			return static_cast<DurationInferenceTaskError *>(task);
		}

		srt::SingerImport findDurationImport(const srt::SingerSpec *singer) {
			for (const auto &import : singer->imports()) {
				if (import.inference()->className() == Dur::API_CLASS) {
					return import;
				}
			}
			return {};
		}

		DSSP_DiffSingerDurationInferenceTask newDurationInferenceTaskError(std::string errorMessage) {
			auto error = std::make_unique<DurationInferenceTaskError>();
			error->errorMessage = std::move(errorMessage);
			auto *handle = error.get();

			std::lock_guard lock(g_durationInferenceTaskMutex);
			g_durationInferenceTaskErrors.emplace(handle, std::move(error));
			return handle;
		}

		void setDurationInferenceImportOptions(
			srt::InferenceSpec *inference,
			srt::NO<srt::InferenceImportOptions> importOptions
		) {
			std::lock_guard lock(g_durationInferenceTaskMutex);
			g_durationInferenceImportOptions[inference] = std::move(importOptions);
		}

		srt::NO<srt::InferenceImportOptions> findDurationInferenceImportOptions(srt::InferenceSpec *inference) {
			std::lock_guard lock(g_durationInferenceTaskMutex);
			if (const auto it = g_durationInferenceImportOptions.find(inference);
				it != g_durationInferenceImportOptions.end()) {
				return it->second;
			}
			return {};
		}

		bool isDurationInferenceTaskError(DSSP_DiffSingerDurationInferenceTask task) {
			std::lock_guard lock(g_durationInferenceTaskMutex);
			return g_durationInferenceTaskErrors.contains(getDiffSingerDurationInferenceTaskError(task));
		}

		const std::string &durationInferenceTaskErrorMessage(DSSP_DiffSingerDurationInferenceTask task) {
			static const std::string empty;

			std::lock_guard lock(g_durationInferenceTaskMutex);
			const auto it = g_durationInferenceTaskErrors.find(getDiffSingerDurationInferenceTaskError(task));
			if (it == g_durationInferenceTaskErrors.end()) {
				return empty;
			}
			return it->second->errorMessage;
		}

		void addDurationInferenceTask(srt::NO<srt::Inference> inference) {
			std::lock_guard lock(g_durationInferenceTaskMutex);
			g_durationInferenceTasks.emplace(inference.get(), std::move(inference));
		}

		srt::NO<srt::Inference> findDurationInferenceTask(srt::Inference *task) {
			std::lock_guard lock(g_durationInferenceTaskMutex);
			if (const auto it = g_durationInferenceTasks.find(task); it != g_durationInferenceTasks.end()) {
				return it->second;
			}
			return {};
		}

		void deleteDurationInferenceTask(DSSP_DiffSingerDurationInferenceTask task) {
			std::lock_guard lock(g_durationInferenceTaskMutex);
			if (const auto it = g_durationInferenceTaskErrors.find(
					getDiffSingerDurationInferenceTaskError(task)
				);
				it != g_durationInferenceTaskErrors.end()) {
				g_durationInferenceTaskErrors.erase(it);
				return;
			}

			g_durationInferenceTasks.erase(getDiffSingerDurationInferenceTask(task));
		}

		DSSP_DiffSingerManagedDoubleArray newDiffSingerManagedDoubleArray(std::vector<double> values) {
			auto result = std::make_unique<DiffSingerManagedDoubleArray>(std::move(values));
			return result.release();
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerDurationInference DSSP_GetDiffSingerDurationInference(DSSP_SRTSinger singer) {
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findDurationImport(singerSpec);
	if (import.isNull()) {
		return nullptr;
	}
	auto *inference = import.inference();
	dssp::setDurationInferenceImportOptions(inference, import.options());
	return inference;
}

const char *DSSP_GetDiffSingerDurationInferenceSpeakerID(DSSP_SRTSinger singer, const char *singer_speaker_id) {
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findDurationImport(singerSpec);
	if (import.isNull()) {
		return singer_speaker_id;
	}

	const auto options = import.options().as<dssp::Dur::DurationImportOptions>();
	if (!options) {
		return singer_speaker_id;
	}
	const auto it = options->speakerMapping.find(singer_speaker_id);
	return it == options->speakerMapping.end() ? singer_speaker_id : it->second.c_str();
}

DSSP_DiffSingerDurationInferenceTask DSSP_CreateDiffSingerDurationInferenceTask(
	DSSP_DiffSingerDurationInference inference
) {
	auto *spec = dssp::getDiffSingerDurationInference(inference);
	if (spec == nullptr) {
		return dssp::newDurationInferenceTaskError("duration inference is nullptr");
	}

	srt::NO<srt::Inference> task;
	auto importOptions = dssp::findDurationInferenceImportOptions(spec);
	if (!importOptions) {
		importOptions = srt::NO<dssp::Dur::DurationImportOptions>::create();
	}
	if (auto exp = spec->createInference(
			importOptions,
			srt::NO<dssp::Dur::DurationRuntimeOptions>::create()
		);
		!exp) {
		return dssp::newDurationInferenceTaskError(exp.error().message());
	} else {
		task = exp.take();
	}

	if (auto exp = task->initialize(srt::NO<dssp::Dur::DurationInitArgs>::create()); !exp) {
		return dssp::newDurationInferenceTaskError(exp.error().message());
	}

	auto *handle = task.get();
	dssp::addDurationInferenceTask(std::move(task));
	dssp::g_logger.info("DiffSinger duration inference task created");
	return handle;
}

void DSSP_DeleteDiffSingerDurationInferenceTask(DSSP_DiffSingerDurationInferenceTask task) {
	dssp::deleteDurationInferenceTask(task);
}

bool DSSP_IsDiffSingerDurationInferenceTaskError(DSSP_DiffSingerDurationInferenceTask task) {
	return dssp::isDurationInferenceTaskError(task);
}

const char *DSSP_GetDiffSingerDurationInferenceErrorMessage(DSSP_DiffSingerDurationInferenceTask task) {
	return dssp::durationInferenceTaskErrorMessage(task).c_str();
}

DSSP_DiffSingerDurationInference DSSP_GetDiffSingerDurationInferenceTaskInference(
	DSSP_DiffSingerDurationInferenceTask task
) {
	if (dssp::isDurationInferenceTaskError(task)) {
		return nullptr;
	}
	const auto *inference = dssp::getDiffSingerDurationInferenceTask(task);
	return const_cast<srt::InferenceSpec *>(inference->spec());
}

DSSP_DiffSingerManagedDoubleArray DSSP_RunDiffSingerDurationInferenceTask(
	DSSP_DiffSingerDurationInferenceTask task,
	double duration,
	DSSP_DiffSingerWords words
) {
	auto *inference = dssp::getDiffSingerDurationInferenceTask(task);
	dssp::g_logger.info("DiffSinger duration inference task started");

	auto input = srt::NO<dssp::Dur::DurationStartInput>::create();
	input->duration = duration;
	input->words = dssp::toDsinferInputWordInfos(*dssp::getDiffSingerWords(words));

	srt::NO<srt::TaskResult> taskResult;
	if (auto exp = inference->start(input); !exp) {
		dssp::g_logger.error(
			std::string("Failed to run DiffSinger duration inference task: ") + exp.error().message()
		);
		return nullptr;
	} else {
		taskResult = exp.take();
	}

	auto durationResult = taskResult.as<dssp::Dur::DurationResult>();
	if (inference->state() == srt::ITask::Failed) {
		dssp::g_logger.error(
			std::string("Failed to run DiffSinger duration inference task: ") + durationResult->error.message()
		);
		return nullptr;
	}
	if (durationResult->durations.empty()) {
		dssp::g_logger.error("Failed to run DiffSinger duration inference task: result is empty");
		return nullptr;
	}

	auto result = dssp::newDiffSingerManagedDoubleArray(std::move(durationResult->durations));
	dssp::g_logger.info("DiffSinger duration inference task completed");
	return result;
}

void DSSP_TerminateDiffSingerDurationInferenceTask(DSSP_DiffSingerDurationInferenceTask task) {
	if (dssp::isDurationInferenceTaskError(task)) {
		return;
	}

	auto inference = dssp::findDurationInferenceTask(dssp::getDiffSingerDurationInferenceTask(task));
	if (!inference) {
		return;
	}
	inference->stop();
	dssp::g_logger.info("DiffSinger duration inference task terminated manually");
}
