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

#include "types.h"

#include <algorithm>
#include <filesystem>
#include <iterator>
#include <memory>
#include <string>

#include <stdcorelib/str.h>
#include <stdcorelib/system.h>

#include <LangCore/Core/Manager.h>
#include <LangCore/Module/Module.h>
#include <LangCore/Task/SessionTask.h>
#include <LangCore/Task/TaskPlugin.h>

namespace {
	std::string g_languageConversionErrorMessage;

	std::filesystem::path getPluginRootDirectory() {
#if defined(__APPLE__)
		return stdc::system::application_directory().parent_path() / _TSTR("PlugIns/LangPlugins");
#else
		return stdc::system::application_directory().parent_path() / _TSTR("lib/plugins/LangPlugins");
#endif
	}

	std::filesystem::path getPackagesRootDirectory() {
#if defined(__APPLE__)
		return stdc::system::application_directory().parent_path() / _TSTR("Resources/G2pPackages");
#else
		return stdc::system::application_directory().parent_path() / _TSTR("share/G2pPackages");
#endif
	}

	LangCore::ExecutionProvider parseExecutionProvider(DSSP_ExecutionProvider ep) {
		if (ep == DSSP_ExecutionProvider_CPU) {
			return LangCore::ExecutionProvider::CPUExecutionProvider;
		}
		if (ep == DSSP_ExecutionProvider_CUDA) {
			return LangCore::ExecutionProvider::CUDAExecutionProvider;
		}
		if (ep == DSSP_ExecutionProvider_DirectML) {
			return LangCore::ExecutionProvider::DMLExecutionProvider;
		}
		if (ep == DSSP_ExecutionProvider_CoreML) {
			return LangCore::ExecutionProvider::CoreMLExecutionProvider;
		}
		return LangCore::ExecutionProvider::CPUExecutionProvider;
	}

	bool initializeONNXDriver(LangCore::Manager *langMgr, DSSP_Device device) {
		const auto onnxDriverPlugin = langMgr->plugin<LangCore::DriverPlugin>("onnx");
		if (!onnxDriverPlugin) {
			// TODO logger
			std::cerr << "Failed to load ONNX inference driver" << std::endl;
			return false;
		}
		const auto expOnnxDriver = onnxDriverPlugin->create();
		if (!expOnnxDriver) {
			// TODO logger
			std::cerr << "Failed to create ONNX inference driver: " << expOnnxDriver.error().message() << std::endl;
			return false;
		}
		const auto onnxDriver = expOnnxDriver.value();
		const auto onnxArgs = LangCore::NO<LangCore::DriverInitArgs>::create();

		const auto ep_ = parseExecutionProvider(DSSP_GetDeviceExecutionProvider(device));
		onnxArgs->ep = ep_;
		const auto ortParentPath = onnxDriverPlugin->path().parent_path() / _TSTR("runtimes") / _TSTR("onnx");
		onnxArgs->runtimePath = ep_ == LangCore::ExecutionProvider::CUDAExecutionProvider ? ortParentPath / _TSTR("cuda") : ortParentPath / _TSTR("default");

		onnxArgs->loadFromProcess = false;
		onnxArgs->deviceIndex = DSSP_GetDeviceIndex(device);

		if (const auto exp = onnxDriver->initialize(onnxArgs); !exp) {
			std::cerr << "Failed to initialize ONNX driver: " << exp.error().message() << std::endl;
			return false;
		}

		auto &driverCategory = *langMgr->category("driver");
		driverCategory.addObject("g2pOnnxDriver", onnxDriver);
		return true;
	}
}

bool DSSP_InitializeLanguageConversion(DSSP_Device device) {
	const auto langMgr = LangCore::Manager::instance();

    const auto defaultPluginDir = getPluginRootDirectory() ;
    langMgr->addPluginPath("org.openvpi.Driver", defaultPluginDir / _TSTR("Drivers"));
    langMgr->addPluginPath("org.openvpi.Task", defaultPluginDir / _TSTR("G2ps"));
    langMgr->addPluginPath("org.openvpi.Task", defaultPluginDir / _TSTR("Taggers"));
    langMgr->addPluginPath("org.openvpi.Task", defaultPluginDir / _TSTR("Splitters"));

    const std::filesystem::path packagesRootDir = getPackagesRootDirectory();
    langMgr->addPackagePath(packagesRootDir);

    initializeONNXDriver(langMgr, device);

	std::string errorMessage;
	langMgr->initialize(errorMessage);
	if (!langMgr->initialized()) {
		g_languageConversionErrorMessage = std::move(errorMessage);
		return false;
	} else {
		g_languageConversionErrorMessage.clear();
		return true;
	}
}

const char *DSSP_GetLanguageConversionErrorMessage(void) {
	return g_languageConversionErrorMessage.c_str();
}

DSSP_Pronunciations DSSP_ConvertLanguage(DSSP_Lyrics lyrics) {
	const auto *input = getLyrics(lyrics);
	auto pronunciations = std::make_unique<Pronunciations>();
	pronunciations->reserve(input->size());
	std::vector<LangCore::G2pInput> g2pInput;
	g2pInput.reserve(input->size());
	std::ranges::transform(*input, std::back_inserter(g2pInput), [](const Lyric &lyric) {
		return LangCore::G2pInput(lyric.text, lyric.language);
	});
	std::vector<LangCore::G2pInput *> g2pInputPtrs;
	g2pInputPtrs.reserve(g2pInput.size());
	std::ranges::transform(g2pInput, std::back_inserter(g2pInputPtrs), [](LangCore::G2pInput &input) {
		return &input;
	});
	auto g2pResults = LangCore::Manager::instance()->convert(g2pInputPtrs);
	std::ranges::transform(g2pResults, std::back_inserter(*pronunciations), [](const LangCore::G2pRes &res) {
		Pronunciation pronunciation;
		pronunciation.text = res.pronunciation;
		pronunciation.candidates = res.candidates;
		pronunciation.isError = res.errorType != LangCore::G2pErrorType::NoError;
		return pronunciation;
	});
	return pronunciations.release();
}
