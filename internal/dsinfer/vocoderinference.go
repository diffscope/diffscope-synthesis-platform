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

type VocoderInference struct {
	handle uintptr
}

type VocoderInferenceTask struct {
	mu         sync.Mutex
	runMu      sync.Mutex
	wg         sync.WaitGroup
	handle     uintptr
	deleted    bool
	deleteDone chan struct{}
	runs       map[*VocoderInferenceRun]struct{}
}

type VocoderInferenceRun struct {
	started   <-chan struct{}
	done      chan VocoderInferenceResult
	terminate func()
}

type VocoderInferenceResult struct {
	AudioData *AudioData
	Err       error
}

var vocoderInferenceQueue = utils.NewSerializedTaskQueue[*AudioData]()

func GetVocoderInference(singer *synthrt.Singer) (*VocoderInference, error) {
	if singer == nil || singer.Handle() == 0 {
		return nil, errors.New("dsinfer: singer is not loaded")
	}

	handle := native.DSSP_GetDiffSingerVocoderInference(singer.Handle())
	if handle == 0 {
		return nil, errors.New("dsinfer: vocoder inference is not available")
	}
	return &VocoderInference{handle: handle}, nil
}

func (i *VocoderInference) Handle() uintptr {
	if i == nil {
		return 0
	}
	return i.handle
}

func (i *VocoderInference) CreateTask() (*VocoderInferenceTask, error) {
	if i == nil || i.handle == 0 {
		return nil, errors.New("dsinfer: vocoder inference is not available")
	}

	taskHandle := native.DSSP_CreateDiffSingerVocoderInferenceTask(i.handle)
	if taskHandle == 0 {
		return nil, vocoderInferenceError("create vocoder inference task", taskHandle)
	}
	if native.DSSP_IsDiffSingerVocoderInferenceTaskError(taskHandle) {
		err := vocoderInferenceError("create vocoder inference task", taskHandle)
		native.DSSP_DeleteDiffSingerVocoderInferenceTask(taskHandle)
		return nil, err
	}
	return &VocoderInferenceTask{
		handle:     taskHandle,
		deleteDone: make(chan struct{}),
		runs:       make(map[*VocoderInferenceRun]struct{}),
	}, nil
}

func (t *VocoderInferenceTask) Handle() uintptr {
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

func (t *VocoderInferenceTask) Start(
	ctx context.Context,
	feature *AcousticFeature,
) (*VocoderInferenceRun, error) {
	if t == nil {
		return nil, errors.New("dsinfer: vocoder inference task is nil")
	}
	if feature == nil || feature.Handle() == 0 {
		return nil, errors.New("dsinfer: vocoder inference acoustic feature is not available")
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
		return nil, errors.New("dsinfer: vocoder inference task is deleted")
	}
	taskHandle := t.handle
	featureHandle := feature.consume()
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			native.DSSP_DeleteDiffSingerAcousticFeature(featureHandle)
		})
	}

	run := &VocoderInferenceRun{
		done: make(chan VocoderInferenceResult, 1),
	}
	t.runs[run] = struct{}{}
	t.wg.Add(1)

	queued := vocoderInferenceQueue.Submit(utils.SerializedTaskSpec[*AudioData]{
		Context: ctx,
		Run: func(context.Context) (*AudioData, error) {
			defer cleanup()

			t.runMu.Lock()
			defer t.runMu.Unlock()
			resultHandle := native.DSSP_RunDiffSingerVocoderInferenceTask(
				taskHandle,
				featureHandle,
			)
			if resultHandle == 0 {
				return nil, vocoderInferenceError("run vocoder inference task", taskHandle)
			}
			return &AudioData{handle: resultHandle}, nil
		},
		Terminate: func() {
			native.DSSP_TerminateDiffSingerVocoderInferenceTask(taskHandle)
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
		run.done <- VocoderInferenceResult{
			AudioData: result.Value,
			Err:       result.Err,
		}
		close(run.done)
	}()

	return run, nil
}

func (t *VocoderInferenceTask) Delete() {
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
	runs := make([]*VocoderInferenceRun, 0, len(t.runs))
	for run := range t.runs {
		runs = append(runs, run)
	}
	t.mu.Unlock()

	for _, run := range runs {
		run.Terminate()
	}
	t.wg.Wait()

	if taskHandle != 0 {
		native.DSSP_DeleteDiffSingerVocoderInferenceTask(taskHandle)
	}

	t.mu.Lock()
	if t.handle == taskHandle {
		t.handle = 0
	}
	t.mu.Unlock()
	close(deleteDone)
}

func (t *VocoderInferenceTask) finishRun(run *VocoderInferenceRun) {
	t.mu.Lock()
	delete(t.runs, run)
	t.mu.Unlock()
	t.wg.Done()
}

func (r *VocoderInferenceRun) Started() <-chan struct{} {
	if r == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return r.started
}

func (r *VocoderInferenceRun) Done() <-chan VocoderInferenceResult {
	if r == nil {
		done := make(chan VocoderInferenceResult, 1)
		done <- VocoderInferenceResult{Err: errors.New("dsinfer: vocoder inference run is nil")}
		close(done)
		return done
	}
	return r.done
}

func (r *VocoderInferenceRun) Terminate() {
	if r == nil || r.terminate == nil {
		return
	}
	r.terminate()
}

func vocoderInferenceError(action string, task uintptr) error {
	if task != 0 && native.DSSP_IsDiffSingerVocoderInferenceTaskError(task) {
		if message := native.DSSP_GetDiffSingerVocoderInferenceErrorMessage(task); message != "" {
			return errors.New(message)
		}
	}
	return fmt.Errorf("dsinfer: %s failed", action)
}
