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

package utils

import (
	"context"
	"errors"
	"sync"
)

// SerializedTaskFunc runs one non-thread-safe task while the queue is held.
type SerializedTaskFunc[T any] func(ctx context.Context) (T, error)

// SerializedTaskTerminateFunc requests cancellation of a running task.
type SerializedTaskTerminateFunc func()

// SerializedTaskCancelFunc releases resources for a task skipped before start.
type SerializedTaskCancelFunc func(err error)

// SerializedTaskSpec describes work submitted to a SerializedTaskQueue.
type SerializedTaskSpec[T any] struct {
	Context   context.Context
	Run       SerializedTaskFunc[T]
	Terminate SerializedTaskTerminateFunc
	Cancel    SerializedTaskCancelFunc
}

// SerializedTaskResult is the terminal result of a serialized task.
type SerializedTaskResult[T any] struct {
	Value T
	Err   error
}

// SerializedTaskQueue runs submitted tasks one at a time in FIFO order.
type SerializedTaskQueue[T any] struct {
	mu    sync.Mutex
	cond  *sync.Cond
	items []*SerializedTask[T]
}

// SerializedTask represents one queued or running task.
type SerializedTask[T any] struct {
	ctx       context.Context
	run       SerializedTaskFunc[T]
	terminate SerializedTaskTerminateFunc
	cancel    SerializedTaskCancelFunc

	mu            sync.Mutex
	started       chan struct{}
	done          chan SerializedTaskResult[T]
	completed     chan struct{}
	state         serializedTaskState
	terminateOnce sync.Once
	cancelOnce    sync.Once
}

type serializedTaskState int

const (
	serializedTaskQueued serializedTaskState = iota
	serializedTaskRunning
	serializedTaskDone
)

var errSerializedTaskRunNil = errors.New("serialized task run function is nil")

// NewSerializedTaskQueue creates a FIFO queue and starts its worker.
func NewSerializedTaskQueue[T any]() *SerializedTaskQueue[T] {
	queue := &SerializedTaskQueue[T]{}
	queue.cond = sync.NewCond(&queue.mu)
	go queue.run()
	return queue
}

// Submit enqueues a task for serialized execution.
func (q *SerializedTaskQueue[T]) Submit(spec SerializedTaskSpec[T]) *SerializedTask[T] {
	ctx := spec.Context
	if ctx == nil {
		ctx = context.Background()
	}
	task := &SerializedTask[T]{
		ctx:       ctx,
		run:       spec.Run,
		terminate: spec.Terminate,
		cancel:    spec.Cancel,
		started:   make(chan struct{}),
		done:      make(chan SerializedTaskResult[T], 1),
		completed: make(chan struct{}),
	}

	go task.watchContext()
	q.mu.Lock()
	q.items = append(q.items, task)
	q.mu.Unlock()
	q.cond.Signal()
	return task
}

// Started is closed when the task has been dequeued and begins processing.
func (t *SerializedTask[T]) Started() <-chan struct{} {
	return t.started
}

// Done receives exactly one terminal result.
func (t *SerializedTask[T]) Done() <-chan SerializedTaskResult[T] {
	return t.done
}

// Terminate cancels a queued task or requests termination of a running task.
func (t *SerializedTask[T]) Terminate() {
	t.cancelQueued(context.Canceled)

	t.mu.Lock()
	running := t.state == serializedTaskRunning
	terminate := t.terminate
	t.mu.Unlock()

	if running && terminate != nil {
		t.terminateOnce.Do(terminate)
	}
}

func (q *SerializedTaskQueue[T]) run() {
	for {
		task := q.next()
		if !task.start() {
			continue
		}
		result := task.runTask()
		task.complete(result)
	}
}

func (q *SerializedTaskQueue[T]) next() *SerializedTask[T] {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait()
	}
	task := q.items[0]
	copy(q.items, q.items[1:])
	q.items[len(q.items)-1] = nil
	q.items = q.items[:len(q.items)-1]
	return task
}

func (t *SerializedTask[T]) start() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state != serializedTaskQueued {
		return false
	}
	if err := t.ctx.Err(); err != nil {
		t.state = serializedTaskDone
		t.cancelOnce.Do(func() {
			if t.cancel != nil {
				t.cancel(err)
			}
		})
		t.done <- SerializedTaskResult[T]{Err: err}
		close(t.done)
		close(t.completed)
		return false
	}
	t.state = serializedTaskRunning
	close(t.started)
	return true
}

func (t *SerializedTask[T]) runTask() SerializedTaskResult[T] {
	if t.run == nil {
		return SerializedTaskResult[T]{Err: errSerializedTaskRunNil}
	}

	watchDone := make(chan struct{})
	go func() {
		select {
		case <-t.ctx.Done():
			t.Terminate()
		case <-watchDone:
		}
	}()

	value, err := t.run(t.ctx)
	close(watchDone)
	if err == nil && t.ctx.Err() != nil {
		err = t.ctx.Err()
	}
	return SerializedTaskResult[T]{
		Value: value,
		Err:   err,
	}
}

func (t *SerializedTask[T]) complete(result SerializedTaskResult[T]) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == serializedTaskDone {
		return
	}
	t.state = serializedTaskDone
	t.done <- result
	close(t.done)
	close(t.completed)
}

func (t *SerializedTask[T]) cancelQueued(err error) {
	t.mu.Lock()
	if t.state != serializedTaskQueued {
		t.mu.Unlock()
		return
	}
	t.state = serializedTaskDone
	t.mu.Unlock()

	t.cancelOnce.Do(func() {
		if t.cancel != nil {
			t.cancel(err)
		}
	})
	t.done <- SerializedTaskResult[T]{Err: err}
	close(t.done)
	close(t.completed)
}

func (t *SerializedTask[T]) watchContext() {
	select {
	case <-t.ctx.Done():
		t.cancelQueued(t.ctx.Err())
	case <-t.completed:
	}
}
