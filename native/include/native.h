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

#ifdef __cplusplus
}
#endif

#endif // DSSP_NATIVE_H
