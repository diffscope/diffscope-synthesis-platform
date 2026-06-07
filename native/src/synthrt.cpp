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

#include "synthrt.h"

#include "logger.h"
#include "nativefileutils.h"

#include <cstdint>
#include <deque>
#include <functional>
#include <memory>
#include <mutex>
#include <stdexcept>
#include <string>
#include <string_view>
#include <unordered_map>
#include <utility>

#include <stdcorelib/support/versionnumber.h>
#include <stdcorelib/system.h>

#include <dsinfer/Api/Drivers/Onnx/OnnxDriverApi.h>
#include <dsinfer/Inference/InferenceDriverPlugin.h>
#include <dsinfer/Core/Tensor.h>

#include <synthrt/Core/Contribute.h>
#include <synthrt/Core/SynthUnit.h>
#include <synthrt/Support/Logging.h>

namespace dssp {

	namespace {
		const dssp::Logger g_logger("native.synthrt");

		void logSynthRTMessage(int level, const std::string &message) {
			if (level <= srt::Logger::Debug) {
				g_logger.debug(message);
				return;
			}
			if (level == srt::Logger::Warning) {
				g_logger.warn(message);
				return;
			}
			if (level >= srt::Logger::Critical) {
				g_logger.error(message);
				return;
			}
			g_logger.info(message);
		}

		void logSynthRT(int level, const srt::LogContext &context, const std::string_view &message) {
			std::string logMessage;
			if (context.category && context.category[0] != '\0') {
				logMessage = std::string(context.category) + ": ";
			}
			logMessage += std::string(message.data(), message.size());
			logSynthRTMessage(level, logMessage);
		}

		struct PackageKey {
			std::string id;
			stdc::VersionNumber version;

			bool operator==(const PackageKey &other) const {
				return id == other.id && version == other.version;
			}
		};

		struct PackageKeyHash {
			size_t operator()(const PackageKey &key) const {
				const auto idHash = std::hash<std::string>()(key.id);
				const auto versionHash = std::hash<stdc::VersionNumber>()(key.version);
				return idHash ^ (versionHash + 0x9e3779b9 + (idHash << 6) + (idHash >> 2));
			}
		};

		std::mutex g_synthRTMutex;
		std::unique_ptr<srt::SynthUnit> g_synthUnit;
		std::string g_synthRTErrorMessage;
		std::deque<srt::PackageRef> g_packages;
		std::unordered_map<PackageKey, std::uintptr_t, PackageKeyHash> g_packageHandles;

		stdc::VersionNumber toVersionNumber(DSSP_SRTVersionNumber versionNumber) {
			return stdc::VersionNumber(
				versionNumber.major,
				versionNumber.minor,
				versionNumber.patch,
				versionNumber.tweak
			);
		}

		PackageKey packageKey(const srt::PackageRef &package) {
			return PackageKey{package.id(), package.version()};
		}

		std::uintptr_t toPackageHandle(DSSP_SRTPackage package) {
			return reinterpret_cast<std::uintptr_t>(package);
		}

		DSSP_SRTPackage fromPackageHandle(std::uintptr_t handle) {
			return reinterpret_cast<DSSP_SRTPackage>(handle);
		}

		srt::SynthUnit *synthUnit() {
			if (!g_synthUnit) {
				throw std::runtime_error("SynthRT is not initialized");
			}
			return g_synthUnit.get();
		}

		std::uintptr_t addPackage(srt::PackageRef package) {
			const auto key = packageKey(package);
			if (const auto it = g_packageHandles.find(key); it != g_packageHandles.end()) {
				return it->second;
			}

			g_packages.push_back(std::move(package));
			const auto handle = static_cast<std::uintptr_t>(g_packages.size());
			g_packageHandles.emplace(packageKey(g_packages.back()), handle);
			return handle;
		}

		std::string packageErrorMessage(const srt::PackageRef &package) {
			const auto error = package.error();
			return error.ok() ? "package is not loaded" : error.message();
		}

		std::uintptr_t getLoadedPackageHandle(
			const char *packageID,
			const stdc::VersionNumber &versionNumber
		) {
			const PackageKey requestedKey{packageID, versionNumber};
			if (const auto it = g_packageHandles.find(requestedKey); it != g_packageHandles.end()) {
				return it->second;
			}

			auto package = synthUnit()->find(packageID, versionNumber);
			if (package.isValid() && package.isLoaded()) {
				return addPackage(std::move(package));
			}
			return 0;
		}

		void setErrorMessage(std::string message) {
			g_synthRTErrorMessage = std::move(message);
		}

		void clearErrorMessage() {
			g_synthRTErrorMessage.clear();
		}

		std::filesystem::path getDsInferPluginsRootDirectory() {
			return stdc::system::application_directory().parent_path() / _TSTR("lib/plugins/dsinfer");
		}

		ds::Api::Onnx::ExecutionProvider parseExecutionProvider(DSSP_ExecutionProvider ep) {
			if (ep == DSSP_ExecutionProvider_CPU) {
				return ds::Api::Onnx::CPUExecutionProvider;
			}
			if (ep == DSSP_ExecutionProvider_CUDA) {
				return ds::Api::Onnx::CUDAExecutionProvider;
			}
			if (ep == DSSP_ExecutionProvider_DirectML) {
				return ds::Api::Onnx::DMLExecutionProvider;
			}
			if (ep == DSSP_ExecutionProvider_CoreML) {
				return ds::Api::Onnx::CoreMLExecutionProvider;
			}
			return ds::Api::Onnx::CPUExecutionProvider;
		}

		bool initializeONNXDriver(srt::SynthUnit *su, DSSP_Device device) {
			const auto onnxDriverPlugin = su->plugin<ds::InferenceDriverPlugin>("onnx");
			if (!onnxDriverPlugin) {
				setErrorMessage("Failed to load ONNX inference driver");
				return false;
			}
			const auto onnxDriver = onnxDriverPlugin->create();
			if (!onnxDriver) {
				setErrorMessage("Failed to create ONNX inference driver");
				return false;
			}
			const auto onnxArgs = srt::NO<ds::Api::Onnx::DriverInitArgs>::create();

			const auto ep_ = parseExecutionProvider(DSSP_GetDeviceExecutionProvider(device));
			onnxArgs->ep = ep_;
			const auto ortParentPath = onnxDriverPlugin->path().parent_path() / _TSTR("runtimes") / _TSTR("onnx");
			onnxArgs->runtimePath = ep_ == ds::Api::Onnx::CUDAExecutionProvider ? ortParentPath / _TSTR("cuda") : ortParentPath / _TSTR("default");
			onnxArgs->deviceIndex = DSSP_GetDeviceIndex(device);

			if (auto exp = onnxDriver->initialize(onnxArgs); !exp) {
				setErrorMessage(std::string("Failed to initialize ONNX driver: ") + exp.error().message());
				return false;
			}

			auto *inferenceCategory = su->category("inference");
			if (!inferenceCategory) {
				setErrorMessage("Failed to load inference category");
				return false;
			}
			inferenceCategory->addObject("dsdriver", onnxDriver);
			return true;
		}

	} // namespace

	srt::PackageRef *getSRTPackage(DSSP_SRTPackage package) {
		const auto handle = toPackageHandle(package);
		if (handle == 0 || handle > g_packages.size()) {
			return nullptr;
		}
		return &g_packages.at(handle - 1);
	}

	srt::SingerSpec *getSRTSinger(DSSP_SRTSinger singer) {
		return static_cast<srt::SingerSpec *>(singer);
	}

} // namespace dssp

bool DSSP_InitializeSynthRT(const char *package_path, DSSP_Device device) {
	std::lock_guard lock(dssp::g_synthRTMutex);
	srt::Logger::setLogCallback(dssp::logSynthRT);

	// TODO: Force the linker to link dsinfer by referencing a symbol from it.
	volatile auto p_ = &ds::Tensor::create;

	if (dssp::g_synthUnit) {
		dssp::g_synthUnit->addPackagePath(dssp::pathFromUtf8(package_path));
		dssp::clearErrorMessage();
		return true;
	}

	auto su = std::make_unique<srt::SynthUnit>();

	const auto defaultPluginDir = dssp::getDsInferPluginsRootDirectory();
	su->addPluginPath("org.openvpi.SingerProvider", defaultPluginDir / _TSTR("singerproviders"));
	su->addPluginPath("org.openvpi.InferenceDriver", defaultPluginDir / _TSTR("inferencedrivers"));
	su->addPluginPath("org.openvpi.InferenceInterpreter", defaultPluginDir / _TSTR("inferenceinterpreters"));

	su->addPackagePath(dssp::pathFromUtf8(package_path));

	if (!dssp::initializeONNXDriver(su.get(), device)) {
		return false;
	}

	dssp::g_synthUnit = std::move(su);
	dssp::clearErrorMessage();
	return true;
}

const char *DSSP_GetSynthRTErrorMessage(void) {
	return dssp::g_synthRTErrorMessage.c_str();
}

DSSP_SRTPackage DSSP_GetSRTPackage(
	const char *package_dir,
	const char *package_id,
	DSSP_SRTVersionNumber versionNumber
) {
	std::lock_guard lock(dssp::g_synthRTMutex);
	const auto version = dssp::toVersionNumber(versionNumber);
	if (const auto handle = dssp::getLoadedPackageHandle(package_id, version); handle != 0) {
		return dssp::fromPackageHandle(handle);
	}

	auto packageResult = dssp::synthUnit()->open(dssp::pathFromUtf8(package_dir), false);
	if (!packageResult) {
		dssp::g_logger.error(packageResult.error().message());
		return nullptr;
	}
	if (!packageResult->isLoaded()) {
		dssp::g_logger.error(dssp::packageErrorMessage(*packageResult));
		return nullptr;
	}

	auto package = packageResult.take();

	const auto handle = dssp::addPackage(std::move(package));
	return dssp::fromPackageHandle(handle);
}

DSSP_SRTSinger DSSP_GetSRTSinger(DSSP_SRTPackage package, const char *singer_id) {
	std::lock_guard lock(dssp::g_synthRTMutex);
	
	auto *packageRef = dssp::getSRTPackage(package);
	auto *contrib = packageRef->contribute("singer", singer_id);
	if (!contrib) {
		dssp::g_logger.error("Singer not found");
		return nullptr;
	}
	auto *singer = contrib->as<srt::SingerSpec>();
	return singer;
}
