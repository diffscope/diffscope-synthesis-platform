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
#include <string>
#include <utility>
#include <vector>

#include <PhonemeConverter/DictionaryS2P.h>
#include <PhonemeConverter/DirectS2P.h>
#include <PhonemeConverter/LuaS2P.h>
#include <PhonemeConverter/LuaScript.h>
#include <PhonemeConverter/MappingS2P.h>

namespace {
	const dssp::Logger g_logger("native.s2p");

	enum class S2PType {
		Error,
		Direct,
		Mapping,
		Dictionary,
		Lua,
	};

	struct S2PHandle {
		S2PType type{S2PType::Error};
		std::string errorMessage;
		std::unique_ptr<PhonemeConverter::DirectS2P> direct;
		std::unique_ptr<PhonemeConverter::MappingS2P> mapping;
		std::unique_ptr<PhonemeConverter::DictionaryS2P> dictionary;
		std::unique_ptr<dssp::LuaRunnerPool<PhonemeConverter::LuaS2P>> lua;
	};

	S2PHandle *getS2P(DSSP_S2P s2p) {
		return static_cast<S2PHandle *>(s2p);
	}

	bool isError(const S2PHandle *s2p) {
		return !s2p->errorMessage.empty();
	}

	void setUnknownError(S2PHandle *s2p) {
		s2p->type = S2PType::Error;
		s2p->errorMessage = "unknown S2P error";
	}

	template <typename Factory>
	DSSP_S2P newS2P(Factory factory) {
		auto s2p = std::make_unique<S2PHandle>();
		try {
			factory(*s2p);
		} catch (const std::exception &e) {
			s2p->type = S2PType::Error;
			s2p->errorMessage = e.what();
		} catch (...) {
			setUnknownError(s2p.get());
		}
		return s2p.release();
	}

	DSSP_Phonemes newEmptyPhonemes() {
		return new Phonemes();
	}

	DSSP_Phonemes makePhonemes(std::vector<std::string> phonemeTexts) {
		auto phonemes = std::make_unique<Phonemes>();
		phonemes->reserve(phonemeTexts.size());
		for (auto &text : phonemeTexts) {
			phonemes->push_back(Phoneme{std::move(text), false});
		}
		return phonemes.release();
	}

	std::vector<std::string> runLuaS2P(S2PHandle *s2p, const char *pronunciationText) {
		auto lease = s2p->lua->acquire();
		if (!lease) {
			return {};
		}
		return (*lease)->convert(pronunciationText);
	}

}

DSSP_S2P DSSP_NewDirectS2P(void) {
	return newS2P([](S2PHandle &s2p) {
		s2p.direct = std::make_unique<PhonemeConverter::DirectS2P>();
		s2p.type = S2PType::Direct;
	});
}

DSSP_S2P DSSP_NewMapS2P(const char *mapping_file_path) {
	return newS2P([mapping_file_path](S2PHandle &s2p) {
		auto stream = dssp::openUtf8File(mapping_file_path);
		s2p.mapping = std::make_unique<PhonemeConverter::MappingS2P>(stream);
		s2p.type = S2PType::Mapping;
	});
}

DSSP_S2P DSSP_NewDictS2P(const char *dictionary_file_path) {
	return newS2P([dictionary_file_path](S2PHandle &s2p) {
		auto stream = dssp::openUtf8File(dictionary_file_path);
		s2p.dictionary = std::make_unique<PhonemeConverter::DictionaryS2P>(stream);
		s2p.type = S2PType::Dictionary;
	});
}

DSSP_S2P DSSP_NewCustomS2P(const char *lua_script_file_path) {
	return newS2P([lua_script_file_path](S2PHandle &s2p) {
		const auto scriptText = dssp::readUtf8File(lua_script_file_path);
		auto script = std::make_shared<PhonemeConverter::LuaScript>(scriptText, lua_script_file_path);
		auto pool = std::make_unique<dssp::LuaRunnerPool<PhonemeConverter::LuaS2P>>(dssp::luaRunnerCount(), [script] {
			return std::make_unique<PhonemeConverter::LuaS2P>(*script);
		});
		s2p.lua = std::move(pool);
		s2p.type = S2PType::Lua;
	});
}

void DSSP_DeleteS2P(DSSP_S2P s2p) {
	delete getS2P(s2p);
}

bool DSSP_IsS2PError(DSSP_S2P s2p) {
	return isError(getS2P(s2p));
}

const char *DSSP_GetS2PErrorMessage(DSSP_S2P s2p) {
	return getS2P(s2p)->errorMessage.c_str();
}

DSSP_Phonemes DSSP_RunS2P(DSSP_S2P s2p, const char *pronunciation_text) {
	auto *handle = getS2P(s2p);
	if (isError(handle)) {
		return newEmptyPhonemes();
	}

	try {
		switch (handle->type) {
			case S2PType::Direct:
				return makePhonemes(PhonemeConverter::DirectS2P::convert(pronunciation_text));
			case S2PType::Mapping:
				return makePhonemes(handle->mapping->convert(pronunciation_text));
			case S2PType::Dictionary:
				return makePhonemes(handle->dictionary->convert(pronunciation_text));
			case S2PType::Lua:
				return makePhonemes(runLuaS2P(handle, pronunciation_text));
			case S2PType::Error:
				break;
		}
	} catch (const std::exception &e) {
		const auto message = std::string("Failed to run S2P: ") + e.what();
		g_logger.error(message);
	} catch (...) {
		g_logger.error("Failed to run S2P: unknown error");
	}

	return newEmptyPhonemes();
}

void DSSP_TerminateCustomS2P(DSSP_S2P s2p) {
	auto *handle = getS2P(s2p);
	if (handle->type == S2PType::Lua && handle->lua) {
		handle->lua->terminate();
	}
}

void DSSP_SetLuaRunnerCount(size_t count) {
	dssp::setLuaRunnerCount(count);
}
