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

#include "native.h"

#include <algorithm>
#include <cctype>
#include <iomanip>
#include <sstream>
#include <string>
#include <vector>

#ifdef _WIN32
#  include <dxgi1_6.h>
#  include <wrl/client.h>
#endif // _WIN32

namespace {

struct DeviceInfo {
	DSSP_ExecutionProvider executionProvider;
	int index;
	std::string description;
	std::string id;
	uint64_t memory;
};

struct ExecutionProviderInfo {
	DSSP_ExecutionProvider executionProvider;
	std::vector<DeviceInfo> devices;
};

#ifdef _WIN32

using Microsoft::WRL::ComPtr;

std::string wstrToString(const wchar_t *wstr) {
	if (wstr == nullptr || *wstr == L'\0') {
		return {};
	}

	const int requiredSize = WideCharToMultiByte(
		CP_UTF8,
		0,
		wstr,
		-1,
		nullptr,
		0,
		nullptr,
		nullptr
	);
	if (requiredSize <= 0) {
		return {};
	}

	std::string result(static_cast<size_t>(requiredSize), '\0');
	const int convertedSize = WideCharToMultiByte(
		CP_UTF8,
		0,
		wstr,
		-1,
		result.data(),
		requiredSize,
		nullptr,
		nullptr
	);
	if (convertedSize <= 0) {
		return {};
	}

	result.pop_back();
	return result;
}

std::string getDmlDeviceId(const DXGI_ADAPTER_DESC1 &desc) {
	std::stringstream ss;
	ss << std::setfill('0') << std::setw(8) << std::hex << desc.VendorId;
	ss << "-";
	ss << std::setfill('0') << std::setw(8) << std::hex << desc.DeviceId;
	return ss.str();
}

std::vector<DeviceInfo> getDmlDevices() {
	std::vector<DeviceInfo> devices;
	ComPtr<IDXGIFactory6> dxgiFactory;
	if (FAILED(CreateDXGIFactory1(IID_PPV_ARGS(&dxgiFactory)))) {
		return devices;
	}

	ComPtr<IDXGIAdapter1> adapter;
	for (int adapterIndex = 0; dxgiFactory->EnumAdapters1(adapterIndex, &adapter) != DXGI_ERROR_NOT_FOUND; ++adapterIndex) {
		DXGI_ADAPTER_DESC1 desc;
		if (FAILED(adapter->GetDesc1(&desc)) || (desc.Flags & DXGI_ADAPTER_FLAG_SOFTWARE)) {
			continue;
		}

		devices.push_back({
			DSSP_ExecutionProvider_DirectML,
			adapterIndex,
			wstrToString(desc.Description),
			getDmlDeviceId(desc),
			desc.DedicatedVideoMemory,
		});
	}

	std::ranges::sort(devices, [](const auto &a, const auto &b) { return a.memory > b.memory; });
	return devices;
}

bool mayBeDedicatedGpu(const DeviceInfo &device) {
	const auto separator = device.id.find('-');
	if (separator == std::string::npos) {
		return false;
	}

	const auto vendorId = std::stoul(device.id.substr(0, separator), nullptr, 16);
	auto description = device.description;
	std::ranges::transform(description, description.begin(), [](unsigned char c) { return std::toupper(c); });
	return vendorId == 0x1002 || vendorId == 0x10DE || vendorId == 0x174B || description == "NVIDIA";
}

#endif // _WIN32

const std::vector<ExecutionProviderInfo> &getExecutionProviders() {
	static const std::vector<ExecutionProviderInfo> executionProviders = [] {
		std::vector<ExecutionProviderInfo> result {
			{
				DSSP_ExecutionProvider_CPU,
				{{DSSP_ExecutionProvider_CPU, 0, {}, {}, 0}},
			},
		};

		// TODO cuda

#ifdef _WIN32
		result.push_back({DSSP_ExecutionProvider_DirectML, getDmlDevices()});
#endif // _WIN32

#ifdef __APPLE__
		result.push_back({
			DSSP_ExecutionProvider_CoreML,
			{{DSSP_ExecutionProvider_CoreML, 0, {}, {}, 0}},
		});
#endif // __APPLE__

		return result;
	}();
	return executionProviders;
}

const ExecutionProviderInfo *getExecutionProvider(DSSP_ExecutionProvider executionProvider) {
	const auto &executionProviders = getExecutionProviders();
	const auto it = std::ranges::find(executionProviders, executionProvider, &ExecutionProviderInfo::executionProvider);
	return it == executionProviders.end() ? nullptr : &*it;
}

const DeviceInfo *getDevice(DSSP_Device device) {
	return static_cast<const DeviceInfo *>(device);
}

const DeviceInfo *getDefaultDevice() {
	static const DeviceInfo *defaultDevice = [] {
		const auto *cpuExecutionProvider = getExecutionProvider(DSSP_ExecutionProvider_CPU);
		const auto *cpuDevice = &cpuExecutionProvider->devices.front();

#ifdef _WIN32
		const auto *dmlExecutionProvider = getExecutionProvider(DSSP_ExecutionProvider_DirectML);
		const DeviceInfo *preferredDevice = nullptr;
		const DeviceInfo *alternativeDevice = nullptr;
		for (const auto &device : dmlExecutionProvider->devices) {
			auto &candidate = mayBeDedicatedGpu(device) ? preferredDevice : alternativeDevice;
			if (candidate == nullptr || device.memory > candidate->memory) {
				candidate = &device;
			}
		}
		if (preferredDevice != nullptr) {
			return preferredDevice;
		}
		if (alternativeDevice != nullptr) {
			return alternativeDevice;
		}
#endif // _WIN32

#ifdef __APPLE__
		return &getExecutionProvider(DSSP_ExecutionProvider_CoreML)->devices.front();
#endif // __APPLE__

		// TODO cuda

		return cpuDevice;
	}();
	return defaultDevice;
}

} // namespace

DSSP_Device DSSP_GetDefaultDevice(void) {
	return const_cast<DeviceInfo *>(getDefaultDevice());
}

DSSP_ExecutionProvider DSSP_GetDeviceExecutionProvider(DSSP_Device device) {
	const auto *deviceInfo = getDevice(device);
	return deviceInfo == nullptr ? DSSP_ExecutionProvider_CPU : deviceInfo->executionProvider;
}

int DSSP_GetDeviceIndex(DSSP_Device device) {
	const auto *deviceInfo = getDevice(device);
	return deviceInfo == nullptr ? 0 : deviceInfo->index;
}

const char *DSSP_GetDeviceDescription(DSSP_Device device) {
	const auto *deviceInfo = getDevice(device);
	return deviceInfo == nullptr ? "" : deviceInfo->description.c_str();
}

const char *DSSP_GetDeviceID(DSSP_Device device) {
	const auto *deviceInfo = getDevice(device);
	return deviceInfo == nullptr ? "" : deviceInfo->id.c_str();
}

uint64_t DSSP_GetDeviceMemory(DSSP_Device device) {
	const auto *deviceInfo = getDevice(device);
	return deviceInfo == nullptr ? 0 : deviceInfo->memory;
}

bool DSSP_HasExecutionProvider(DSSP_ExecutionProvider execution_provider) {
	return getExecutionProvider(execution_provider) != nullptr;
}

size_t DSSP_GetExecutionProviderDeviceCount(DSSP_ExecutionProvider execution_provider) {
	const auto *executionProvider = getExecutionProvider(execution_provider);
	return executionProvider == nullptr ? 0 : executionProvider->devices.size();
}

DSSP_Device DSSP_GetExecutionProviderDevice(DSSP_ExecutionProvider execution_provider, size_t index) {
	const auto *executionProvider = getExecutionProvider(execution_provider);
	if (executionProvider == nullptr || index >= executionProvider->devices.size()) {
		return nullptr;
	}
	return const_cast<DeviceInfo *>(&executionProvider->devices[index]);
}
