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

#ifndef DSSP_DSINFERINFERENCEHELPER_H
#define DSSP_DSINFERINFERENCEHELPER_H

#include "logger.h"
#include "synthrt.h"

#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <utility>

#include <synthrt/SVS/Inference.h>
#include <synthrt/SVS/InferenceContrib.h>
#include <synthrt/SVS/SingerContrib.h>

namespace dssp {

	template<
		const dssp::Logger *logger,
		const char *apiClass,
		const char *displayName,
		typename ImportOptions,
		typename RuntimeOptions,
		typename InitArgs>
	class DSInferInferenceHelper {
	public:
		static srt::InferenceSpec *inference(void *handle) {
			return static_cast<srt::InferenceSpec *>(handle);
		}

		static srt::Inference *task(void *handle) {
			return static_cast<srt::Inference *>(handle);
		}

		static srt::SingerImport findImport(const srt::SingerSpec *singer) {
			for (const auto &import : singer->imports()) {
				if (import.inference()->className() == apiClass) {
					return import;
				}
			}
			return {};
		}

		static std::string getInferenceFullID(const srt::InferenceSpec *spec) {
			return spec->parent().id() + "@" + spec->parent().version().toString() + ":" + spec->id();
		}

		static std::string getTaskInferenceFullID(const srt::Inference *task) {
			return getInferenceFullID(task->spec());
		}

		static void *getInference(DSSP_SRTSinger singer) {
			const auto *singerSpec = getSRTSinger(singer);
			const auto import = findImport(singerSpec);
			if (import.isNull()) {
				return nullptr;
			}

			auto *spec = import.inference();
			setImportOptions(spec, import.options());
			return spec;
		}

		static const char *speakerID(DSSP_SRTSinger singer, const char *singerSpeakerID) {
			const auto *singerSpec = getSRTSinger(singer);
			const auto import = findImport(singerSpec);
			if (import.isNull()) {
				return singerSpeakerID;
			}

			const auto options = import.options().template as<ImportOptions>();
			if (!options) {
				return singerSpeakerID;
			}
			const auto it = options->speakerMapping.find(singerSpeakerID);
			return it == options->speakerMapping.end() ? singerSpeakerID : it->second.c_str();
		}

		static void *createTask(void *inferenceHandle) {
			auto *spec = inference(inferenceHandle);
			if (spec == nullptr) {
				return newTaskError(std::string(displayName) + " inference is nullptr");
			}

			srt::NO<srt::Inference> inferenceTask;
			auto importOptions = findImportOptions(spec);
			if (!importOptions) {
				importOptions = srt::NO<ImportOptions>::create();
			}
			if (auto exp = spec->createInference(
					importOptions,
					srt::NO<RuntimeOptions>::create()
				);
				!exp) {
				return newTaskError(exp.error().message());
			} else {
				inferenceTask = exp.take();
			}

			if (auto exp = inferenceTask->initialize(srt::NO<InitArgs>::create()); !exp) {
				return newTaskError(exp.error().message());
			}

			auto *handle = inferenceTask.get();
			const auto inferenceFullID = getInferenceFullID(spec);
			addTask(std::move(inferenceTask));
			logger->infoF("DiffSinger {} inference task created: {}", displayName, inferenceFullID);
			return handle;
		}

		static void deleteTask(void *taskHandle) {
			bool deletedTaskError = false;
			std::string inferenceFullID;

			{
				std::lock_guard lock(taskMutex);
				if (const auto it = taskErrors.find(taskError(taskHandle)); it != taskErrors.end()) {
					taskErrors.erase(it);
					deletedTaskError = true;
				} else if (const auto it = tasks.find(task(taskHandle)); it != tasks.end()) {
					inferenceFullID = getTaskInferenceFullID(it->second.get());
					tasks.erase(it);
				}
			}

			if (deletedTaskError) {
				logger->infoF("DiffSinger {} inference task error deleted", displayName);
			} else if (!inferenceFullID.empty()) {
				logger->infoF("DiffSinger {} inference task deleted: {}", displayName, inferenceFullID);
			}
		}

		static bool isTaskError(void *taskHandle) {
			std::lock_guard lock(taskMutex);
			return taskErrors.contains(taskError(taskHandle));
		}

		static const std::string &taskErrorMessage(void *taskHandle) {
			static const std::string empty;

			std::lock_guard lock(taskMutex);
			const auto it = taskErrors.find(taskError(taskHandle));
			if (it == taskErrors.end()) {
				return empty;
			}
			return it->second->errorMessage;
		}

		static void *taskInference(void *taskHandle) {
			if (isTaskError(taskHandle)) {
				return nullptr;
			}
			const auto *inferenceTask = task(taskHandle);
			return const_cast<srt::InferenceSpec *>(inferenceTask->spec());
		}

		static srt::NO<srt::Inference> findTask(void *taskHandle) {
			std::lock_guard lock(taskMutex);
			if (const auto it = tasks.find(task(taskHandle)); it != tasks.end()) {
				return it->second;
			}
			return {};
		}

		static srt::NO<srt::TaskResult> runTask(
			void *taskHandle,
			const srt::NO<srt::TaskStartInput> &input
		) {
			auto *inferenceTask = task(taskHandle);
			const auto inferenceFullID = getTaskInferenceFullID(inferenceTask);
			logger->infoF("DiffSinger {} inference task started: {}", displayName, inferenceFullID);

			srt::NO<srt::TaskResult> taskResult;
			if (auto exp = inferenceTask->start(input); !exp) {
				logger->errorF(
					"Failed to run DiffSinger {} inference task ({}): {}",
					displayName,
					inferenceFullID,
					exp.error().message()
				);
				return {};
			} else {
				taskResult = exp.take();
			}

			if (inferenceTask->state() == srt::ITask::Failed) {
				logger->errorF(
					"Failed to run DiffSinger {} inference task ({})",
					displayName,
					inferenceFullID
				);
				return {};
			}

			logger->infoF("DiffSinger {} inference task completed: {}", displayName, inferenceFullID);
			return taskResult;
		}

		static void terminateTask(void *taskHandle) {
			if (isTaskError(taskHandle)) {
				return;
			}

			auto inferenceTask = findTask(taskHandle);
			if (!inferenceTask) {
				return;
			}
			const auto inferenceFullID = getTaskInferenceFullID(inferenceTask.get());
			inferenceTask->stop();
			logger->infoF(
				"DiffSinger {} inference task terminated manually: {}",
				displayName,
				inferenceFullID
			);
		}

	private:
		struct TaskError {
			std::string errorMessage;
		};

		static TaskError *taskError(void *handle) {
			return static_cast<TaskError *>(handle);
		}

		static void *newTaskError(std::string errorMessage) {
			auto error = std::make_unique<TaskError>();
			error->errorMessage = std::move(errorMessage);
			auto *handle = error.get();

			std::lock_guard lock(taskMutex);
			taskErrors.emplace(handle, std::move(error));
			return handle;
		}

		static void setImportOptions(
			srt::InferenceSpec *inference,
			srt::NO<srt::InferenceImportOptions> options
		) {
			std::lock_guard lock(taskMutex);
			importOptions[inference] = std::move(options);
		}

		static srt::NO<srt::InferenceImportOptions> findImportOptions(srt::InferenceSpec *inference) {
			std::lock_guard lock(taskMutex);
			if (const auto it = importOptions.find(inference); it != importOptions.end()) {
				return it->second;
			}
			return {};
		}

		static void addTask(srt::NO<srt::Inference> inferenceTask) {
			std::lock_guard lock(taskMutex);
			tasks.emplace(inferenceTask.get(), std::move(inferenceTask));
		}

		inline static std::mutex taskMutex;
		inline static std::unordered_map<srt::InferenceSpec *, srt::NO<srt::InferenceImportOptions>>
			importOptions;
		inline static std::unordered_map<srt::Inference *, srt::NO<srt::Inference>> tasks;
		inline static std::unordered_map<TaskError *, std::unique_ptr<TaskError>> taskErrors;
	};

} // namespace dssp

#endif // DSSP_DSINFERINFERENCEHELPER_H
