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

package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var requestValidator = newRequestValidator()

func newRequestValidator() *validator.Validate {
	validate := validator.New()
	validate.RegisterStructValidation(validateDurationContext, durationContext{})
	return validate
}

func decodeRequest(c *gin.Context, value any) error {
	if err := decodeJSON(c, value); err != nil {
		return err
	}
	return requestValidator.Struct(value)
}

func decodeJSON(c *gin.Context, value any) error {
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func writeBadRequest(c *gin.Context) {
	if c.Request.Context().Err() != nil {
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"message": ""})
}

func validateDurationContext(sl validator.StructLevel) {
	var context durationContext
	switch current := sl.Current().Interface().(type) {
	case durationContext:
		context = current
	case *durationContext:
		if current == nil {
			return
		}
		context = *current
	default:
		return
	}

	if context.Singers == nil || context.Mix == nil {
		return
	}

	expectedLength := len(context.Singers) - 1
	if expectedLength < 0 {
		return
	}
	for _, mix := range context.Mix {
		if !isValidMix(mix, expectedLength) {
			sl.ReportError(context.Mix, "Mix", "Mix", "duration_mix", "")
			return
		}
	}
}

func isValidMix(mix []float64, expectedLength int) bool {
	if mix == nil || len(mix) != expectedLength {
		return false
	}
	var sum float64
	for _, value := range mix {
		if value < 0 || value > 1 {
			return false
		}
		sum += value
	}
	return sum >= 0 && sum <= 1
}
