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

package dsinfer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"diffscope-synthesis-platform/internal/synthrt"
	"diffscope-synthesis-platform/internal/utils"
	"diffscope-synthesis-platform/native"
)

type DurationInference struct {
	handle uintptr
}

type DurationInferenceTask struct {
	started   <-chan struct{}
	done      <-chan DurationInferenceResult
	terminate func()
}

type DurationInferenceResult struct {
	Durations []float64
	Err       error
}

var durationInferenceQueue = utils.NewSerializedTaskQueue[[]float64]()

func NewDurationInference(singer *synthrt.Singer) (*DurationInference, error) {
	if singer == nil || singer.Handle() == 0 {
		return nil, errors.New("dsinfer: singer is not loaded")
	}

	handle := native.DSSP_GetDiffSingerDurationInference(singer.Handle())
	if handle == 0 {
		return nil, errors.New("dsinfer: duration inference is not available")
	}
	return &DurationInference{handle: handle}, nil
}

func GetDurationInferenceSpeakerID(singer *synthrt.Singer, singerSpeakerID string) (string, error) {
	if singer == nil || singer.Handle() == 0 {
		return "", errors.New("dsinfer: singer is not loaded")
	}
	return native.DSSP_GetDiffSingerDurationInferenceSpeakerID(singer.Handle(), singerSpeakerID), nil
}

func (i *DurationInference) Handle() uintptr {
	if i == nil {
		return 0
	}
	return i.handle
}

func (i *DurationInference) Start(ctx context.Context, duration float64, words *Words) (*DurationInferenceTask, error) {
	if i == nil || i.handle == 0 {
		return nil, errors.New("dsinfer: duration inference is not available")
	}
	if words == nil || words.Handle() == 0 {
		return nil, errors.New("dsinfer: duration inference words are not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	taskHandle := native.DSSP_CreateDiffSingerDurationInferenceTask(i.handle)
	if taskHandle == 0 {
		return nil, durationInferenceError("create duration inference task", taskHandle)
	}
	if native.DSSP_IsDiffSingerDurationInferenceTaskError(taskHandle) {
		err := durationInferenceError("create duration inference task", taskHandle)
		native.DSSP_DeleteDiffSingerDurationInferenceTask(taskHandle)
		return nil, err
	}

	wordsHandle := words.consume()
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			native.DSSP_DeleteDiffSingerDurationInferenceTask(taskHandle)
			native.DSSP_FreeDiffSingerWords(wordsHandle)
		})
	}

	queued := durationInferenceQueue.Submit(utils.SerializedTaskSpec[[]float64]{
		Context: ctx,
		Run: func(context.Context) ([]float64, error) {
			defer cleanup()

			resultHandle := native.DSSP_RunDiffSingerDurationInferenceTask(taskHandle, duration, wordsHandle)
			if resultHandle == 0 {
				return nil, durationInferenceError("run duration inference task", taskHandle)
			}
			result := (&ManagedDoubleArray{handle: resultHandle}).Values()
			native.DSSP_FreeDiffSingerManagedDoubleArray(resultHandle)
			return result, nil
		},
		Terminate: func() {
			native.DSSP_TerminateDiffSingerDurationInferenceTask(taskHandle)
		},
		Cancel: func(error) {
			cleanup()
		},
	})

	done := make(chan DurationInferenceResult, 1)
	go func() {
		result := <-queued.Done()
		done <- DurationInferenceResult{
			Durations: result.Value,
			Err:       result.Err,
		}
		close(done)
	}()

	return &DurationInferenceTask{
		started: queued.Started(),
		done:    done,
		terminate: func() {
			queued.Terminate()
		},
	}, nil
}

func (t *DurationInferenceTask) Started() <-chan struct{} {
	if t == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return t.started
}

func (t *DurationInferenceTask) Done() <-chan DurationInferenceResult {
	if t == nil {
		done := make(chan DurationInferenceResult, 1)
		done <- DurationInferenceResult{Err: errors.New("dsinfer: duration inference task is nil")}
		close(done)
		return done
	}
	return t.done
}

func (t *DurationInferenceTask) Terminate() {
	if t == nil || t.terminate == nil {
		return
	}
	t.terminate()
}

func durationInferenceError(action string, task uintptr) error {
	if task != 0 && native.DSSP_IsDiffSingerDurationInferenceTaskError(task) {
		if message := native.DSSP_GetDiffSingerDurationInferenceErrorMessage(task); message != "" {
			return errors.New(message)
		}
	}
	return fmt.Errorf("dsinfer: %s failed", action)
}
