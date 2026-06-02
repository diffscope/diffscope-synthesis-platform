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

typedef enum DSSP_LanguageConversionError {
	DSSP_LanguageConversionError_None,
	DSSP_LanguageConversionError_InternalError,
} DSSP_LanguageConversionError;

typedef struct DSSP_LanguageConversionResult {
	DSSP_Pronunciations pronunciations;
	DSSP_LanguageConversionError error;
} DSSP_LanguageConversionResult;

DSSP_LanguageConversionResult DSSP_ConvertLanguage(DSSP_Lyrics lyrics);

#ifdef __cplusplus
}
#endif

#endif // DSSP_NATIVE_H
