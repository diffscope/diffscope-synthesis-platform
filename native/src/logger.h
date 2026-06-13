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

#ifndef DSSP_LOGGER_H
#define DSSP_LOGGER_H

#include "native.h"

#include <format>
#include <string>
#include <string_view>
#include <utility>

namespace dssp {

	class Logger {
	public:
		explicit Logger(const char *component);

		void debug(const std::string &message) const;
		void info(const std::string &message) const;
		void warn(const std::string &message) const;
		void error(const std::string &message) const;

		template<typename... Args>
		void debugF(const std::string_view format, Args&&... args) const {
			debug(std::vformat(format, std::make_format_args(args...)));
		}

		template<typename... Args>
		void infoF(const std::string_view format, Args&&... args) const {
			info(std::vformat(format, std::make_format_args(args...)));
		}

		template<typename... Args>
		void warnF(const std::string_view format, Args&&... args) const {
			warn(std::vformat(format, std::make_format_args(args...)));
		}

		template<typename... Args>
		void errorF(const std::string_view format, Args&&... args) const {
			error(std::vformat(format, std::make_format_args(args...)));
		}

	private:
		void log(int level, const std::string &message) const;

		const char *_component;
	};

}

#endif // DSSP_LOGGER_H
