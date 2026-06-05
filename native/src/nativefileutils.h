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

#ifndef DSSP_NATIVE_FILE_UTILS_H
#define DSSP_NATIVE_FILE_UTILS_H

#include <filesystem>
#include <fstream>
#include <ios>
#include <iterator>
#include <stdexcept>
#include <string>

namespace dssp {

	inline std::filesystem::path pathFromUtf8(const char *path) {
		if (!path) {
			throw std::invalid_argument("path must not be null");
		}

#if defined(_WIN32)
		const std::string pathString(path);
		std::u8string u8Path;
		u8Path.reserve(pathString.size());
		for (const auto ch : pathString) {
			u8Path.push_back(static_cast<char8_t>(static_cast<unsigned char>(ch)));
		}
		return std::filesystem::path(u8Path);
#else
		return std::filesystem::path(path);
#endif
	}

	inline std::ifstream openUtf8File(const char *path, std::ios::openmode mode = std::ios::in) {
		std::ifstream stream(pathFromUtf8(path), mode);
		if (!stream) {
			throw std::runtime_error("failed to open file: " + std::string(path));
		}
		return stream;
	}

	inline std::string readUtf8File(const char *path) {
		auto stream = openUtf8File(path, std::ios::in | std::ios::binary);
		return std::string(std::istreambuf_iterator<char>(stream), std::istreambuf_iterator<char>());
	}

}

#endif // DSSP_NATIVE_FILE_UTILS_H
