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

#include <dsinfer/Api/Inferences/Pitch/1/PitchApiL1.h>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	namespace Pit = ds::Api::Pitch::L1;

	namespace {
		const dssp::Logger g_logger("native.dsinferpitch");

		struct PitchInferenceTaskError {
			std::string errorMessage;
		};

		std::mutex g_pitchInferenceTaskMutex;
		std::unordered_map<srt::InferenceSpec *, srt::NO<srt::InferenceImportOptions>>
			g_pitchInferenceImportOptions;
		std::unordered_map<srt::Inference *, srt::NO<srt::Inference>> g_pitchInferenceTasks;
		std::unordered_map<PitchInferenceTaskError *, std::unique_ptr<PitchInferenceTaskError>>
			g_pitchInferenceTaskErrors;

		srt::InferenceSpec *getDiffSingerPitchInference(DSSP_DiffSingerPitchInference inference) {
			return static_cast<srt::InferenceSpec *>(inference);
		}

		srt::Inference *getDiffSingerPitchInferenceTask(DSSP_DiffSingerPitchInferenceTask task) {
			return static_cast<srt::Inference *>(task);
		}

		PitchInferenceTaskError *getDiffSingerPitchInferenceTaskError(
			DSSP_DiffSingerPitchInferenceTask task
		) {
			return static_cast<PitchInferenceTaskError *>(task);
		}

		srt::SingerImport findPitchImport(const srt::SingerSpec *singer) {
			for (const auto &import : singer->imports()) {
				if (import.inference()->className() == Pit::API_CLASS) {
					return import;
				}
			}
			return {};
		}

		DSSP_DiffSingerPitchInferenceTask newPitchInferenceTaskError(std::string errorMessage) {
			auto error = std::make_unique<PitchInferenceTaskError>();
			error->errorMessage = std::move(errorMessage);
			auto *handle = error.get();

			std::lock_guard lock(g_pitchInferenceTaskMutex);
			g_pitchInferenceTaskErrors.emplace(handle, std::move(error));
			return handle;
		}

		void setPitchInferenceImportOptions(
			srt::InferenceSpec *inference,
			srt::NO<srt::InferenceImportOptions> importOptions
		) {
			std::lock_guard lock(g_pitchInferenceTaskMutex);
			g_pitchInferenceImportOptions[inference] = std::move(importOptions);
		}

		srt::NO<srt::InferenceImportOptions> findPitchInferenceImportOptions(srt::InferenceSpec *inference) {
			std::lock_guard lock(g_pitchInferenceTaskMutex);
			if (const auto it = g_pitchInferenceImportOptions.find(inference);
				it != g_pitchInferenceImportOptions.end()) {
				return it->second;
			}
			return {};
		}

		bool isPitchInferenceTaskError(DSSP_DiffSingerPitchInferenceTask task) {
			std::lock_guard lock(g_pitchInferenceTaskMutex);
			return g_pitchInferenceTaskErrors.contains(getDiffSingerPitchInferenceTaskError(task));
		}

		const std::string &pitchInferenceTaskErrorMessage(DSSP_DiffSingerPitchInferenceTask task) {
			static const std::string empty;

			std::lock_guard lock(g_pitchInferenceTaskMutex);
			const auto it = g_pitchInferenceTaskErrors.find(getDiffSingerPitchInferenceTaskError(task));
			if (it == g_pitchInferenceTaskErrors.end()) {
				return empty;
			}
			return it->second->errorMessage;
		}

		void addPitchInferenceTask(srt::NO<srt::Inference> inference) {
			std::lock_guard lock(g_pitchInferenceTaskMutex);
			g_pitchInferenceTasks.emplace(inference.get(), std::move(inference));
		}

		srt::NO<srt::Inference> findPitchInferenceTask(srt::Inference *task) {
			std::lock_guard lock(g_pitchInferenceTaskMutex);
			if (const auto it = g_pitchInferenceTasks.find(task); it != g_pitchInferenceTasks.end()) {
				return it->second;
			}
			return {};
		}

		void deletePitchInferenceTask(DSSP_DiffSingerPitchInferenceTask task) {
			std::lock_guard lock(g_pitchInferenceTaskMutex);
			if (const auto it = g_pitchInferenceTaskErrors.find(
					getDiffSingerPitchInferenceTaskError(task)
				);
				it != g_pitchInferenceTaskErrors.end()) {
				g_pitchInferenceTaskErrors.erase(it);
				return;
			}

			g_pitchInferenceTasks.erase(getDiffSingerPitchInferenceTask(task));
		}

		DSSP_DiffSingerManagedDoubleArray newDiffSingerManagedDoubleArray(std::vector<double> values) {
			auto result = std::make_unique<DiffSingerManagedDoubleArray>(std::move(values));
			return result.release();
		}

	} // namespace

} // namespace dssp

DSSP_DiffSingerPitchInference DSSP_GetDiffSingerPitchInference(DSSP_SRTSinger singer) {
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findPitchImport(singerSpec);
	if (import.isNull()) {
		return nullptr;
	}
	auto *inference = import.inference();
	dssp::setPitchInferenceImportOptions(inference, import.options());
	return inference;
}

const char *DSSP_GetDiffSingerPitchInferenceSpeakerID(DSSP_SRTSinger singer, const char *singer_speaker_id) {
	const auto *singerSpec = dssp::getSRTSinger(singer);
	const auto import = dssp::findPitchImport(singerSpec);
	if (import.isNull()) {
		return singer_speaker_id;
	}

	const auto options = import.options().as<dssp::Pit::PitchImportOptions>();
	if (!options) {
		return singer_speaker_id;
	}
	const auto it = options->speakerMapping.find(singer_speaker_id);
	return it == options->speakerMapping.end() ? singer_speaker_id : it->second.c_str();
}

DSSP_DiffSingerPitchInferenceTask DSSP_CreateDiffSingerPitchInferenceTask(
	DSSP_DiffSingerPitchInference inference
) {
	auto *spec = dssp::getDiffSingerPitchInference(inference);
	if (spec == nullptr) {
		return dssp::newPitchInferenceTaskError("pitch inference is nullptr");
	}

	srt::NO<srt::Inference> task;
	auto importOptions = dssp::findPitchInferenceImportOptions(spec);
	if (!importOptions) {
		importOptions = srt::NO<dssp::Pit::PitchImportOptions>::create();
	}
	if (auto exp = spec->createInference(
			importOptions,
			srt::NO<dssp::Pit::PitchRuntimeOptions>::create()
		);
		!exp) {
		return dssp::newPitchInferenceTaskError(exp.error().message());
	} else {
		task = exp.take();
	}

	if (auto exp = task->initialize(srt::NO<dssp::Pit::PitchInitArgs>::create()); !exp) {
		return dssp::newPitchInferenceTaskError(exp.error().message());
	}

	auto *handle = task.get();
	dssp::addPitchInferenceTask(std::move(task));
	dssp::g_logger.info("DiffSinger pitch inference task created");
	return handle;
}

void DSSP_DeleteDiffSingerPitchInferenceTask(DSSP_DiffSingerPitchInferenceTask task) {
	dssp::deletePitchInferenceTask(task);
}

bool DSSP_IsDiffSingerPitchInferenceTaskError(DSSP_DiffSingerPitchInferenceTask task) {
	return dssp::isPitchInferenceTaskError(task);
}

const char *DSSP_GetDiffSingerPitchInferenceErrorMessage(DSSP_DiffSingerPitchInferenceTask task) {
	return dssp::pitchInferenceTaskErrorMessage(task).c_str();
}

DSSP_DiffSingerPitchInference DSSP_GetDiffSingerPitchInferenceTaskInference(
	DSSP_DiffSingerPitchInferenceTask task
) {
	if (dssp::isPitchInferenceTaskError(task)) {
		return nullptr;
	}
	const auto *inference = dssp::getDiffSingerPitchInferenceTask(task);
	return const_cast<srt::InferenceSpec *>(inference->spec());
}

DSSP_DiffSingerManagedDoubleArray DSSP_RunDiffSingerPitchInferenceTask(
	DSSP_DiffSingerPitchInferenceTask task,
	double duration,
	DSSP_DiffSingerWords words,
	DSSP_DiffSingerParameters parameters,
	DSSP_DiffSingerDynamicMixedSpeakers dynamicMixedSpeakers,
	int64_t steps
) {
	auto *inference = dssp::getDiffSingerPitchInferenceTask(task);
	dssp::g_logger.info("DiffSinger pitch inference task started");

	auto input = srt::NO<dssp::Pit::PitchStartInput>::create();
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
			std::string("Failed to run DiffSinger pitch inference task: ") + exp.error().message()
		);
		return nullptr;
	} else {
		taskResult = exp.take();
	}

	auto pitchResult = taskResult.as<dssp::Pit::PitchResult>();
	if (inference->state() == srt::ITask::Failed) {
		dssp::g_logger.error(
			std::string("Failed to run DiffSinger pitch inference task: ") + pitchResult->error.message()
		);
		return nullptr;
	}
	if (pitchResult->pitch.empty()) {
		dssp::g_logger.error("Failed to run DiffSinger pitch inference task: result is empty");
		return nullptr;
	}

	auto result = dssp::newDiffSingerManagedDoubleArray(std::move(pitchResult->pitch));
	dssp::g_logger.info("DiffSinger pitch inference task completed");
	return result;
}

void DSSP_TerminateDiffSingerPitchInferenceTask(DSSP_DiffSingerPitchInferenceTask task) {
	if (dssp::isPitchInferenceTaskError(task)) {
		return;
	}

	auto inference = dssp::findPitchInferenceTask(dssp::getDiffSingerPitchInferenceTask(task));
	if (!inference) {
		return;
	}
	inference->stop();
	dssp::g_logger.info("DiffSinger pitch inference task terminated manually");
}
