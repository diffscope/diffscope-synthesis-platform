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

type VarianceInference struct {
	handle uintptr
}

type VarianceInferenceTask struct {
	mu         sync.Mutex
	runMu      sync.Mutex
	wg         sync.WaitGroup
	handle     uintptr
	deleted    bool
	deleteDone chan struct{}
	runs       map[*VarianceInferenceRun]struct{}
}

type VarianceInferenceRun struct {
	started   <-chan struct{}
	done      chan VarianceInferenceResult
	terminate func()
}

type VarianceInferenceResult struct {
	Parameters []Parameter
	Err        error
}

var varianceInferenceQueue = utils.NewSerializedTaskQueue[[]Parameter]()

func GetVarianceInference(singer *synthrt.Singer) (*VarianceInference, error) {
	if singer == nil || singer.Handle() == 0 {
		return nil, errors.New("dsinfer: singer is not loaded")
	}

	handle := native.DSSP_GetDiffSingerVarianceInference(singer.Handle())
	if handle == 0 {
		return nil, errors.New("dsinfer: variance inference is not available")
	}
	return &VarianceInference{handle: handle}, nil
}

func GetVarianceInferenceSpeakerID(singer *synthrt.Singer, singerSpeakerID string) (string, error) {
	if singer == nil || singer.Handle() == 0 {
		return "", errors.New("dsinfer: singer is not loaded")
	}
	return native.DSSP_GetDiffSingerVarianceInferenceSpeakerID(singer.Handle(), singerSpeakerID), nil
}

func (i *VarianceInference) Handle() uintptr {
	if i == nil {
		return 0
	}
	return i.handle
}

func (i *VarianceInference) CreateTask() (*VarianceInferenceTask, error) {
	if i == nil || i.handle == 0 {
		return nil, errors.New("dsinfer: variance inference is not available")
	}

	taskHandle := native.DSSP_CreateDiffSingerVarianceInferenceTask(i.handle)
	if taskHandle == 0 {
		return nil, varianceInferenceError("create variance inference task", taskHandle)
	}
	if native.DSSP_IsDiffSingerVarianceInferenceTaskError(taskHandle) {
		err := varianceInferenceError("create variance inference task", taskHandle)
		native.DSSP_DeleteDiffSingerVarianceInferenceTask(taskHandle)
		return nil, err
	}
	return &VarianceInferenceTask{
		handle:     taskHandle,
		deleteDone: make(chan struct{}),
		runs:       make(map[*VarianceInferenceRun]struct{}),
	}, nil
}

func (t *VarianceInferenceTask) Handle() uintptr {
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

func (t *VarianceInferenceTask) Start(
	ctx context.Context,
	duration float64,
	words *Words,
	parameters *Parameters,
	dynamicMixedSpeakers *DynamicMixedSpeakers,
	steps int64,
) (*VarianceInferenceRun, error) {
	if t == nil {
		return nil, errors.New("dsinfer: variance inference task is nil")
	}
	if words == nil || words.Handle() == 0 {
		return nil, errors.New("dsinfer: variance inference words are not available")
	}
	if parameters == nil || parameters.Handle() == 0 {
		return nil, errors.New("dsinfer: variance inference parameters are not available")
	}
	if dynamicMixedSpeakers == nil || dynamicMixedSpeakers.Handle() == 0 {
		return nil, errors.New("dsinfer: variance inference dynamic mixed speakers are not available")
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
		return nil, errors.New("dsinfer: variance inference task is deleted")
	}
	taskHandle := t.handle
	wordsHandle := words.consume()
	parametersHandle := parameters.consume()
	dynamicMixedSpeakersHandle := dynamicMixedSpeakers.consume()
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			native.DSSP_FreeDiffSingerWords(wordsHandle)
			native.DSSP_FreeDiffSingerParameters(parametersHandle)
			native.DSSP_FreeDiffSingerDynamicMixedSpeakers(dynamicMixedSpeakersHandle)
		})
	}

	run := &VarianceInferenceRun{
		done: make(chan VarianceInferenceResult, 1),
	}
	t.runs[run] = struct{}{}
	t.wg.Add(1)

	queued := varianceInferenceQueue.Submit(utils.SerializedTaskSpec[[]Parameter]{
		Context: ctx,
		Run: func(context.Context) ([]Parameter, error) {
			defer cleanup()

			t.runMu.Lock()
			defer t.runMu.Unlock()
			resultHandle := native.DSSP_RunDiffSingerVarianceInferenceTask(
				taskHandle,
				duration,
				wordsHandle,
				parametersHandle,
				dynamicMixedSpeakersHandle,
				steps,
			)
			if resultHandle == 0 {
				return nil, varianceInferenceError("run variance inference task", taskHandle)
			}
			result := (&Parameters{handle: resultHandle}).Values()
			native.DSSP_FreeDiffSingerParameters(resultHandle)
			return result, nil
		},
		Terminate: func() {
			native.DSSP_TerminateDiffSingerVarianceInferenceTask(taskHandle)
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
		run.done <- VarianceInferenceResult{
			Parameters: result.Value,
			Err:        result.Err,
		}
		close(run.done)
	}()

	return run, nil
}

func (t *VarianceInferenceTask) Delete() {
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
	runs := make([]*VarianceInferenceRun, 0, len(t.runs))
	for run := range t.runs {
		runs = append(runs, run)
	}
	t.mu.Unlock()

	for _, run := range runs {
		run.Terminate()
	}
	t.wg.Wait()

	if taskHandle != 0 {
		native.DSSP_DeleteDiffSingerVarianceInferenceTask(taskHandle)
	}

	t.mu.Lock()
	if t.handle == taskHandle {
		t.handle = 0
	}
	t.mu.Unlock()
	close(deleteDone)
}

func (t *VarianceInferenceTask) finishRun(run *VarianceInferenceRun) {
	t.mu.Lock()
	delete(t.runs, run)
	t.mu.Unlock()
	t.wg.Done()
}

func (r *VarianceInferenceRun) Started() <-chan struct{} {
	if r == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return r.started
}

func (r *VarianceInferenceRun) Done() <-chan VarianceInferenceResult {
	if r == nil {
		done := make(chan VarianceInferenceResult, 1)
		done <- VarianceInferenceResult{Err: errors.New("dsinfer: variance inference run is nil")}
		close(done)
		return done
	}
	return r.done
}

func (r *VarianceInferenceRun) Terminate() {
	if r == nil || r.terminate == nil {
		return
	}
	r.terminate()
}

func varianceInferenceError(action string, task uintptr) error {
	if task != 0 && native.DSSP_IsDiffSingerVarianceInferenceTaskError(task) {
		if message := native.DSSP_GetDiffSingerVarianceInferenceErrorMessage(task); message != "" {
			return errors.New(message)
		}
	}
	return fmt.Errorf("dsinfer: %s failed", action)
}
