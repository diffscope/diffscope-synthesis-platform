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

type AcousticInference struct {
	handle uintptr
}

type AcousticInferenceTask struct {
	mu         sync.Mutex
	runMu      sync.Mutex
	wg         sync.WaitGroup
	handle     uintptr
	deleted    bool
	deleteDone chan struct{}
	runs       map[*AcousticInferenceRun]struct{}
}

type AcousticInferenceRun struct {
	started   <-chan struct{}
	done      chan AcousticInferenceResult
	terminate func()
}

type AcousticInferenceResult struct {
	Feature *AcousticFeature
	Err     error
}

var acousticInferenceQueue = utils.NewSerializedTaskQueue[*AcousticFeature]()

func GetAcousticInference(singer *synthrt.Singer) (*AcousticInference, error) {
	if singer == nil || singer.Handle() == 0 {
		return nil, errors.New("dsinfer: singer is not loaded")
	}

	handle := native.DSSP_GetDiffSingerAcousticInference(singer.Handle())
	if handle == 0 {
		return nil, errors.New("dsinfer: acoustic inference is not available")
	}
	return &AcousticInference{handle: handle}, nil
}

func GetAcousticInferenceSpeakerID(singer *synthrt.Singer, singerSpeakerID string) (string, error) {
	if singer == nil || singer.Handle() == 0 {
		return "", errors.New("dsinfer: singer is not loaded")
	}
	return native.DSSP_GetDiffSingerAcousticInferenceSpeakerID(singer.Handle(), singerSpeakerID), nil
}

func (i *AcousticInference) Handle() uintptr {
	if i == nil {
		return 0
	}
	return i.handle
}

func (i *AcousticInference) CreateTask() (*AcousticInferenceTask, error) {
	if i == nil || i.handle == 0 {
		return nil, errors.New("dsinfer: acoustic inference is not available")
	}

	taskHandle := native.DSSP_CreateDiffSingerAcousticInferenceTask(i.handle)
	if taskHandle == 0 {
		return nil, acousticInferenceError("create acoustic inference task", taskHandle)
	}
	if native.DSSP_IsDiffSingerAcousticInferenceTaskError(taskHandle) {
		err := acousticInferenceError("create acoustic inference task", taskHandle)
		native.DSSP_DeleteDiffSingerAcousticInferenceTask(taskHandle)
		return nil, err
	}
	return &AcousticInferenceTask{
		handle:     taskHandle,
		deleteDone: make(chan struct{}),
		runs:       make(map[*AcousticInferenceRun]struct{}),
	}, nil
}

func (t *AcousticInferenceTask) Handle() uintptr {
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

func (t *AcousticInferenceTask) Start(
	ctx context.Context,
	duration float64,
	words *Words,
	parameters *Parameters,
	dynamicMixedSpeakers *DynamicMixedSpeakers,
	depth float32,
	steps int64,
) (*AcousticInferenceRun, error) {
	if t == nil {
		return nil, errors.New("dsinfer: acoustic inference task is nil")
	}
	if words == nil || words.Handle() == 0 {
		return nil, errors.New("dsinfer: acoustic inference words are not available")
	}
	if parameters == nil || parameters.Handle() == 0 {
		return nil, errors.New("dsinfer: acoustic inference parameters are not available")
	}
	if dynamicMixedSpeakers == nil || dynamicMixedSpeakers.Handle() == 0 {
		return nil, errors.New("dsinfer: acoustic inference dynamic mixed speakers are not available")
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
		return nil, errors.New("dsinfer: acoustic inference task is deleted")
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

	run := &AcousticInferenceRun{
		done: make(chan AcousticInferenceResult, 1),
	}
	t.runs[run] = struct{}{}
	t.wg.Add(1)

	queued := acousticInferenceQueue.Submit(utils.SerializedTaskSpec[*AcousticFeature]{
		Context: ctx,
		Run: func(context.Context) (*AcousticFeature, error) {
			defer cleanup()

			t.runMu.Lock()
			defer t.runMu.Unlock()
			resultHandle := native.DSSP_RunDiffSingerAcousticInferenceTask(
				taskHandle,
				duration,
				wordsHandle,
				parametersHandle,
				dynamicMixedSpeakersHandle,
				depth,
				steps,
			)
			if resultHandle == 0 {
				return nil, acousticInferenceError("run acoustic inference task", taskHandle)
			}
			return &AcousticFeature{handle: resultHandle}, nil
		},
		Terminate: func() {
			native.DSSP_TerminateDiffSingerAcousticInferenceTask(taskHandle)
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
		if result.Err != nil && result.Value != nil {
			result.Value.Close()
			result.Value = nil
		}
		run.done <- AcousticInferenceResult{
			Feature: result.Value,
			Err:     result.Err,
		}
		close(run.done)
	}()

	return run, nil
}

func (t *AcousticInferenceTask) Delete() {
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
	runs := make([]*AcousticInferenceRun, 0, len(t.runs))
	for run := range t.runs {
		runs = append(runs, run)
	}
	t.mu.Unlock()

	for _, run := range runs {
		run.Terminate()
	}
	t.wg.Wait()

	if taskHandle != 0 {
		native.DSSP_DeleteDiffSingerAcousticInferenceTask(taskHandle)
	}

	t.mu.Lock()
	if t.handle == taskHandle {
		t.handle = 0
	}
	t.mu.Unlock()
	close(deleteDone)
}

func (t *AcousticInferenceTask) finishRun(run *AcousticInferenceRun) {
	t.mu.Lock()
	delete(t.runs, run)
	t.mu.Unlock()
	t.wg.Done()
}

func (r *AcousticInferenceRun) Started() <-chan struct{} {
	if r == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return r.started
}

func (r *AcousticInferenceRun) Done() <-chan AcousticInferenceResult {
	if r == nil {
		done := make(chan AcousticInferenceResult, 1)
		done <- AcousticInferenceResult{Err: errors.New("dsinfer: acoustic inference run is nil")}
		close(done)
		return done
	}
	return r.done
}

func (r *AcousticInferenceRun) Terminate() {
	if r == nil || r.terminate == nil {
		return
	}
	r.terminate()
}

func acousticInferenceError(action string, task uintptr) error {
	if task != 0 && native.DSSP_IsDiffSingerAcousticInferenceTaskError(task) {
		if message := native.DSSP_GetDiffSingerAcousticInferenceErrorMessage(task); message != "" {
			return errors.New(message)
		}
	}
	return fmt.Errorf("dsinfer: %s failed", action)
}
