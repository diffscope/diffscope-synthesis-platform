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

package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ConsoleHandler struct {
	mu     *sync.Mutex
	w      io.Writer
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
}

func NewConsoleHandler(w io.Writer, level slog.Leveler) *ConsoleHandler {
	if level == nil {
		level = slog.LevelInfo
	}

	return &ConsoleHandler{
		mu:    &sync.Mutex{},
		w:     w,
		level: level,
	}
}

func (h *ConsoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *ConsoleHandler) Handle(_ context.Context, r slog.Record) error {
	var attrs []slog.Attr

	attrs = append(attrs, h.attrs...)

	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	component := "default"
	var fields []string

	for _, a := range attrs {
		appendAttr(&fields, &component, nil, a)
	}

	timestamp := r.Time.Format("2006-01-02T15:04:05.000 -07:00")
	level := r.Level.String()

	var b strings.Builder
	fmt.Fprintf(&b, "[%s] [%s] [%s]: %s", timestamp, component, level, r.Message)

	if len(fields) > 0 {
		b.WriteByte(' ')
		b.WriteByte('[')
		b.WriteString(strings.Join(fields, " "))
		b.WriteByte(']')
	}

	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.w.Write([]byte(b.String()))
	return err
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &nh
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	nh := *h
	if name != "" {
		nh.groups = append(append([]string{}, h.groups...), name)
	}
	return &nh
}

func appendAttr(fields *[]string, component *string, groups []string, a slog.Attr) {
	a.Value = a.Value.Resolve()

	if a.Equal(slog.Attr{}) {
		return
	}

	key := a.Key
	if len(groups) > 0 {
		key = strings.Join(append(groups, key), ".")
	}

	if a.Value.Kind() == slog.KindGroup {
		nextGroups := groups
		if a.Key != "" {
			nextGroups = append(nextGroups, a.Key)
		}

		for _, ga := range a.Value.Group() {
			appendAttr(fields, component, nextGroups, ga)
		}
		return
	}

	if key == "component" {
		*component = valueToString(a.Value)
		return
	}

	*fields = append(*fields, key+"="+valueToString(a.Value))
}

func valueToString(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return quoteIfNeeded(v.String())
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'f', -1, 64)
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339Nano)
	default:
		return quoteIfNeeded(fmt.Sprint(v.Any()))
	}
}

func quoteIfNeeded(s string) string {
	if s == "" || strings.ContainsAny(s, " \t\n\r=[]") {
		return strconv.Quote(s)
	}
	return s
}

func init() {
	handler := NewConsoleHandler(os.Stderr, slog.LevelDebug)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}
