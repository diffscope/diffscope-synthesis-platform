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

package api

type ErrorCode string

const (
	ErrorCodeInternalError       ErrorCode = "INTERNAL_ERROR"
	ErrorCodeUnknownArch         ErrorCode = "UNKNOWN_ARCH"
	ErrorCodeSingerNotExist      ErrorCode = "SINGER_NOT_EXIST"
	ErrorCodeSingerConfigInvalid ErrorCode = "SINGER_CONFIG_INVALID"
	ErrorCodeInvalidParameter    ErrorCode = "INVALID_PARAMETER"
	ErrorCodeSingersUnmixable    ErrorCode = "SINGERS_UNMIXABLE"
)

type Error struct {
	Code    ErrorCode
	Message string
}

func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}
