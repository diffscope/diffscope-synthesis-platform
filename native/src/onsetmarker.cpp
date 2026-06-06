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
#include "luarunnerpool.h"
#include "nativefileutils.h"
#include "types.h"

#include <exception>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include <PhonemeConverter/LuaOnsetMarker.h>
#include <PhonemeConverter/LuaScript.h>
#include <PhonemeConverter/RuleOnsetMarker.h>

namespace dssp {

	namespace {
		const dssp::Logger g_logger("native.onsetmarker");

		enum class OnsetMarkerType {
			Error,
			Rule,
			Lua,
		};

		struct OnsetMarkerHandle {
			OnsetMarkerType type{OnsetMarkerType::Error};
			std::string errorMessage;
			std::unique_ptr<PhonemeConverter::RuleOnsetMarker> rule;
			std::unique_ptr<dssp::LuaRunnerPool<PhonemeConverter::LuaOnsetMarker>> lua;
		};

		OnsetMarkerHandle *getOnsetMarker(DSSP_OnsetMarker onsetMarker) {
			return static_cast<OnsetMarkerHandle *>(onsetMarker);
		}

		bool isError(const OnsetMarkerHandle *onsetMarker) {
			return !onsetMarker->errorMessage.empty();
		}

		void setUnknownError(OnsetMarkerHandle *onsetMarker) {
			onsetMarker->type = OnsetMarkerType::Error;
			onsetMarker->errorMessage = "unknown onset marker error";
		}

		template <typename Factory>
		DSSP_OnsetMarker newOnsetMarker(Factory factory) {
			auto onsetMarker = std::make_unique<OnsetMarkerHandle>();
			try {
				factory(*onsetMarker);
			} catch (const std::exception &e) {
				onsetMarker->type = OnsetMarkerType::Error;
				onsetMarker->errorMessage = e.what();
			} catch (...) {
				setUnknownError(onsetMarker.get());
			}
			return onsetMarker.release();
		}

		std::vector<std::string> phonemeTexts(const Phonemes *phonemes) {
			std::vector<std::string> texts;
			texts.reserve(phonemes->size());
			for (const auto &phoneme : *phonemes) {
				texts.push_back(phoneme.text);
			}
			return texts;
		}

		std::optional<std::vector<bool>> runLuaOnsetMarker(OnsetMarkerHandle *onsetMarker, const std::vector<std::string> &texts) {
			auto lease = onsetMarker->lua->acquire();
			if (!lease) {
				return std::nullopt;
			}
			return (*lease)->mark(texts);
		}

	} // namespace

} // namespace dssp

DSSP_OnsetMarker DSSP_NewRuleOnsetMarker(const char *rule_file_path) {
	return dssp::newOnsetMarker([rule_file_path](dssp::OnsetMarkerHandle &onsetMarker) {
		auto stream = dssp::openUtf8File(rule_file_path);
		onsetMarker.rule = std::make_unique<PhonemeConverter::RuleOnsetMarker>(stream);
		onsetMarker.type = dssp::OnsetMarkerType::Rule;
	});
}

DSSP_OnsetMarker DSSP_NewCustomOnsetMarker(const char *lua_script_file_path) {
	return dssp::newOnsetMarker([lua_script_file_path](dssp::OnsetMarkerHandle &onsetMarker) {
		const auto scriptText = dssp::readUtf8File(lua_script_file_path);
		auto script = std::make_shared<PhonemeConverter::LuaScript>(scriptText, lua_script_file_path);
		auto pool = std::make_unique<dssp::LuaRunnerPool<PhonemeConverter::LuaOnsetMarker>>(dssp::luaRunnerCount(), [script] {
			return std::make_unique<PhonemeConverter::LuaOnsetMarker>(*script);
		});
		onsetMarker.lua = std::move(pool);
		onsetMarker.type = dssp::OnsetMarkerType::Lua;
	});
}

void DSSP_DeleteOnsetMarker(DSSP_OnsetMarker onset_marker) {
	delete dssp::getOnsetMarker(onset_marker);
}

bool DSSP_IsOnsetMarkerError(DSSP_OnsetMarker onset_marker) {
	return dssp::isError(dssp::getOnsetMarker(onset_marker));
}

const char *DSSP_GetOnsetMarkerErrorMessage(DSSP_OnsetMarker onset_marker) {
	return dssp::getOnsetMarker(onset_marker)->errorMessage.c_str();
}

void DSSP_RunOnsetMarker(DSSP_OnsetMarker onset_marker, DSSP_Phonemes phonemes) {
	auto *handle = dssp::getOnsetMarker(onset_marker);
	if (dssp::isError(handle)) {
		return;
	}

	auto *input = dssp::getPhonemes(phonemes);
	const auto texts = dssp::phonemeTexts(input);

	try {
		std::optional<std::vector<bool>> isOnset;
		switch (handle->type) {
			case dssp::OnsetMarkerType::Rule:
				isOnset = handle->rule->mark(texts);
				break;
			case dssp::OnsetMarkerType::Lua:
				isOnset = dssp::runLuaOnsetMarker(handle, texts);
				break;
			case dssp::OnsetMarkerType::Error:
				break;
		}

		if (!isOnset) {
			return;
		}
		if (isOnset->size() != input->size()) {
			dssp::g_logger.error("Failed to run onset marker: result size does not match input size");
			return;
		}

		for (std::size_t i = 0; i < input->size(); ++i) {
			input->at(i).isOnset = isOnset->at(i);
		}
	} catch (const std::exception &e) {
		const auto message = std::string("Failed to run onset marker: ") + e.what();
		dssp::g_logger.error(message);
	} catch (...) {
		dssp::g_logger.error("Failed to run onset marker: unknown error");
	}
}

void DSSP_TerminateCustomOnsetMarker(DSSP_OnsetMarker onset_marker) {
	auto *handle = dssp::getOnsetMarker(onset_marker);
	if (handle->type == dssp::OnsetMarkerType::Lua && handle->lua) {
		handle->lua->terminate();
	}
}
