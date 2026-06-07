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
	mu         sync.Mutex
	runMu      sync.Mutex
	wg         sync.WaitGroup
	handle     uintptr
	deleted    bool
	deleteDone chan struct{}
	runs       map[*DurationInferenceRun]struct{}
}

type DurationInferenceRun struct {
	started   <-chan struct{}
	done      chan DurationInferenceResult
	terminate func()
}

type DurationInferenceResult struct {
	Durations []float64
	Err       error
}

var durationInferenceQueue = utils.NewSerializedTaskQueue[[]float64]()

func GetDurationInference(singer *synthrt.Singer) (*DurationInference, error) {
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

func (i *DurationInference) CreateTask() (*DurationInferenceTask, error) {
	if i == nil || i.handle == 0 {
		return nil, errors.New("dsinfer: duration inference is not available")
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
	return &DurationInferenceTask{
		handle:     taskHandle,
		deleteDone: make(chan struct{}),
		runs:       make(map[*DurationInferenceRun]struct{}),
	}, nil
}

func (t *DurationInferenceTask) Handle() uintptr {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.deleted {
		return 0
	}
	return t.handle
}

func (t *DurationInferenceTask) Start(ctx context.Context, duration float64, words *Words) (*DurationInferenceRun, error) {
	if t == nil {
		return nil, errors.New("dsinfer: duration inference task is nil")
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

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.deleted || t.handle == 0 {
		return nil, errors.New("dsinfer: duration inference task is deleted")
	}
	taskHandle := t.handle
	wordsHandle := words.consume()
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			native.DSSP_FreeDiffSingerWords(wordsHandle)
		})
	}

	run := &DurationInferenceRun{
		done: make(chan DurationInferenceResult, 1),
	}
	t.runs[run] = struct{}{}
	t.wg.Add(1)

	queued := durationInferenceQueue.Submit(utils.SerializedTaskSpec[[]float64]{
		Context: ctx,
		Run: func(context.Context) ([]float64, error) {
			defer cleanup()

			t.runMu.Lock()
			defer t.runMu.Unlock()
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

	run.started = queued.Started()
	run.terminate = queued.Terminate
	go func() {
		defer t.finishRun(run)

		result := <-queued.Done()
		run.done <- DurationInferenceResult{
			Durations: result.Value,
			Err:       result.Err,
		}
		close(run.done)
	}()

	return run, nil
}

func (t *DurationInferenceTask) Delete() {
	if t == nil {
		return
	}

	t.mu.Lock()
	if t.deleteDone == nil {
		t.deleteDone = make(chan struct{})
	}
	if t.deleted {
		deleteDone := t.deleteDone
		t.mu.Unlock()
		<-deleteDone
		return
	}
	t.deleted = true
	taskHandle := t.handle
	deleteDone := t.deleteDone
	runs := make([]*DurationInferenceRun, 0, len(t.runs))
	for run := range t.runs {
		runs = append(runs, run)
	}
	t.mu.Unlock()

	for _, run := range runs {
		run.Terminate()
	}
	t.wg.Wait()

	if taskHandle != 0 {
		native.DSSP_DeleteDiffSingerDurationInferenceTask(taskHandle)
	}

	t.mu.Lock()
	if t.handle == taskHandle {
		t.handle = 0
	}
	t.mu.Unlock()
	close(deleteDone)
}

func (t *DurationInferenceTask) finishRun(run *DurationInferenceRun) {
	t.mu.Lock()
	delete(t.runs, run)
	t.mu.Unlock()
	t.wg.Done()
}

func (r *DurationInferenceRun) Started() <-chan struct{} {
	if r == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return r.started
}

func (r *DurationInferenceRun) Done() <-chan DurationInferenceResult {
	if r == nil {
		done := make(chan DurationInferenceResult, 1)
		done <- DurationInferenceResult{Err: errors.New("dsinfer: duration inference run is nil")}
		close(done)
		return done
	}
	return r.done
}

func (r *DurationInferenceRun) Terminate() {
	if r == nil || r.terminate == nil {
		return
	}
	r.terminate()
}

func durationInferenceError(action string, task uintptr) error {
	if task != 0 && native.DSSP_IsDiffSingerDurationInferenceTaskError(task) {
		if message := native.DSSP_GetDiffSingerDurationInferenceErrorMessage(task); message != "" {
			return errors.New(message)
		}
	}
	return fmt.Errorf("dsinfer: %s failed", action)
}
