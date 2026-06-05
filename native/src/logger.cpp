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

#include "logger.h"

namespace {
	constexpr auto kLogLevelDebug = -4;
	constexpr auto kLogLevelInfo = 0;
	constexpr auto kLogLevelWarning = 4;
	constexpr auto kLogLevelError = 8;

	DSSP_LogCallback g_logCallback = nullptr;
}

void DSSP_SetLogCallback(DSSP_LogCallback log_callback) {
	g_logCallback = log_callback;
}

namespace dssp {

	Logger::Logger(const char *component) : _component(component) {
	}

	void Logger::debug(const std::string &message) const {
		log(kLogLevelDebug, message);
	}

	void Logger::info(const std::string &message) const {
		log(kLogLevelInfo, message);
	}

	void Logger::warn(const std::string &message) const {
		log(kLogLevelWarning, message);
	}

	void Logger::error(const std::string &message) const {
		log(kLogLevelError, message);
	}

	void Logger::log(int level, const std::string &message) const {
		if (!g_logCallback) {
			return;
		}
		g_logCallback(_component, level, message.c_str());
	}

}
