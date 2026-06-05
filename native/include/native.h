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

#ifndef DSSP_NATIVE_H
#define DSSP_NATIVE_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* ========================================================================
 * Logging
 * ====================================================================== */

typedef void (*DSSP_LogCallback)(const char *component, int level, const char *message);

void DSSP_SetLogCallback(DSSP_LogCallback log_callback);

/* ========================================================================
 * Execution Provider Info
 * ====================================================================== */

typedef enum DSSP_ExecutionProvider {
	DSSP_ExecutionProvider_CPU,
	DSSP_ExecutionProvider_CUDA,
	DSSP_ExecutionProvider_DirectML,
	DSSP_ExecutionProvider_CoreML,
} DSSP_ExecutionProvider;

typedef void *DSSP_Device;

DSSP_Device DSSP_GetDefaultDevice(void);
DSSP_ExecutionProvider DSSP_GetDeviceExecutionProvider(DSSP_Device device);
int DSSP_GetDeviceIndex(DSSP_Device device);
const char *DSSP_GetDeviceDescription(DSSP_Device device);
const char *DSSP_GetDeviceID(DSSP_Device device);
uint64_t DSSP_GetDeviceMemory(DSSP_Device device);
bool DSSP_HasExecutionProvider(DSSP_ExecutionProvider execution_provider);
size_t DSSP_GetExecutionProviderDeviceCount(DSSP_ExecutionProvider execution_provider);
DSSP_Device DSSP_GetExecutionProviderDevice(DSSP_ExecutionProvider execution_provider, size_t index);

/* ========================================================================
 * Language Conversion
 * ====================================================================== */

typedef void *DSSP_Lyrics;

DSSP_Lyrics DSSP_AllocateLyrics(size_t count);
void DSSP_FreeLyrics(DSSP_Lyrics lyrics);
size_t DSSP_GetLyricCount(DSSP_Lyrics lyrics);
void DSSP_SetLyricText(DSSP_Lyrics lyrics, size_t index, const char *text);
void DSSP_SetLyricLanguage(DSSP_Lyrics lyrics, size_t index, const char *language);
const char *DSSP_GetLyricText(DSSP_Lyrics lyrics, size_t index);
const char *DSSP_GetLyricLanguage(DSSP_Lyrics lyrics, size_t index);

typedef void *DSSP_Pronunciations;

void DSSP_FreePronunciations(DSSP_Pronunciations pronunciations);
size_t DSSP_GetPronunciationCount(DSSP_Pronunciations pronunciations);
const char *DSSP_GetPronunciationText(DSSP_Pronunciations pronunciations, size_t index);
size_t DSSP_GetPronunciationCandidateCount(DSSP_Pronunciations pronunciations, size_t index);
const char *DSSP_GetPronunciationCandidate(DSSP_Pronunciations pronunciations, size_t index, size_t candidate_index);
bool DSSP_IsPronunciationError(DSSP_Pronunciations pronunciations, size_t index);

bool DSSP_InitializeLanguageConversion(DSSP_Device device);
const char *DSSP_GetLanguageConversionErrorMessage(void);
DSSP_Pronunciations DSSP_ConvertLanguage(DSSP_Lyrics lyrics);

/* ========================================================================
 * Phoneme Conversion
 * ====================================================================== */

typedef void *DSSP_Phonemes;

void DSSP_FreePhonemes(DSSP_Phonemes phonemes);
size_t DSSP_GetPhonemeCount(DSSP_Phonemes phonemes);
const char *DSSP_GetPhonemeText(DSSP_Phonemes phonemes, size_t index);
bool DSSP_IsPhonemeOnset(DSSP_Phonemes phonemes, size_t index);

typedef void *DSSP_S2P;

DSSP_S2P DSSP_NewDirectS2P(void);
DSSP_S2P DSSP_NewMapS2P(const char *mapping_file_path);
DSSP_S2P DSSP_NewDictS2P(const char *dictionary_file_path);
DSSP_S2P DSSP_NewCustomS2P(const char *lua_script_file_path);
void DSSP_DeleteS2P(DSSP_S2P s2p);
bool DSSP_IsS2PError(DSSP_S2P s2p);
const char *DSSP_GetS2PErrorMessage(DSSP_S2P s2p);

// thread-safe
DSSP_Phonemes DSSP_RunS2P(DSSP_S2P s2p, const char *pronunciation_text);
void DSSP_TerminateCustomS2P(DSSP_S2P s2p);

typedef void *DSSP_OnsetMarker;

DSSP_OnsetMarker DSSP_NewRuleOnsetMarker(const char *rule_file_path);
DSSP_OnsetMarker DSSP_NewCustomOnsetMarker(const char *lua_script_file_path);
void DSSP_DeleteOnsetMarker(DSSP_OnsetMarker onset_marker);
bool DSSP_IsOnsetMarkerError(DSSP_OnsetMarker onset_marker);
const char *DSSP_GetOnsetMarkerErrorMessage(DSSP_OnsetMarker onset_marker);

// thread-safe
void DSSP_RunOnsetMarker(DSSP_OnsetMarker onset_marker, DSSP_Phonemes phonemes);
void DSSP_TerminateCustomOnsetMarker(DSSP_OnsetMarker onset_marker);

void DSSP_SetLuaRunnerCount(size_t count);

#ifdef __cplusplus
}
#endif

#endif // DSSP_NATIVE_H
