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
	"sync"
	"time"
)

// DestroyFunc releases a resource after it has been removed from a manager.
type DestroyFunc[K comparable, V any] func(key K, value V)

// ResourceManager manages keyed resources with timeout-based cleanup and
// lease-style reference counting.
type ResourceManager[K comparable, V any] struct {
	mu sync.Mutex

	timeout            time.Duration
	scanInterval       time.Duration
	customScanInterval bool
	destroy            DestroyFunc[K, V]
	resources          map[K]*resourceEntry[K, V]
}

// ResourceLease keeps a resource alive until Release is called.
type ResourceLease[K comparable, V any] struct {
	manager *ResourceManager[K, V]
	entry   *resourceEntry[K, V]

	releaseOnce sync.Once
}

type resourceEntry[K comparable, V any] struct {
	key            K
	value          V
	lastAccessTime time.Time
	refCount       int
	pendingDestroy bool
	destroyed      bool
}

// NewResourceManager creates a ResourceManager and starts its cleanup loop.
func NewResourceManager[K comparable, V any](
	timeout time.Duration,
	scanInterval time.Duration,
	destroy DestroyFunc[K, V],
) *ResourceManager[K, V] {
	manager := &ResourceManager[K, V]{
		timeout:            timeout,
		scanInterval:       normalizeScanInterval(timeout, scanInterval),
		customScanInterval: scanInterval > 0,
		destroy:            destroy,
		resources:          make(map[K]*resourceEntry[K, V]),
	}
	go manager.sweepLoop()
	return manager
}

// SetTimeout updates the resource timeout. A non-positive timeout disables
// timeout-based cleanup.
func (m *ResourceManager[K, V]) SetTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.timeout = timeout
	if !m.customScanInterval {
		m.scanInterval = normalizeScanInterval(timeout, 0)
	}
}

// SetScanInterval updates the background cleanup interval. A non-positive
// interval restores the default interval derived from the current timeout.
func (m *ResourceManager[K, V]) SetScanInterval(scanInterval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.customScanInterval = scanInterval > 0
	m.scanInterval = normalizeScanInterval(m.timeout, scanInterval)
}

// Put stores a resource. If the key already exists, the old resource is
// destroyed immediately or after its outstanding leases are released.
func (m *ResourceManager[K, V]) Put(key K, value V) {
	now := time.Now()

	var destroyItem resourceDestroyItem[K, V]
	m.mu.Lock()
	if oldEntry, ok := m.resources[key]; ok {
		delete(m.resources, key)
		destroyItem = oldEntry.markForDestroyLocked()
	}
	m.resources[key] = &resourceEntry[K, V]{
		key:            key,
		value:          value,
		lastAccessTime: now,
	}
	m.mu.Unlock()

	m.destroyResource(destroyItem)
}

// Get returns a resource and refreshes its last access time.
func (m *ResourceManager[K, V]) Get(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.resources[key]
	if !ok {
		var zero V
		return zero, false
	}
	entry.lastAccessTime = time.Now()
	return entry.value, true
}

// Acquire returns a lease for a resource and refreshes its last access time.
func (m *ResourceManager[K, V]) Acquire(key K) (*ResourceLease[K, V], bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.resources[key]
	if !ok {
		return nil, false
	}
	entry.lastAccessTime = time.Now()
	entry.refCount++
	return &ResourceLease[K, V]{
		manager: m,
		entry:   entry,
	}, true
}

// Delete removes a resource from the manager. If the resource is leased, it is
// destroyed after the final lease is released.
func (m *ResourceManager[K, V]) Delete(key K) bool {
	var destroyItem resourceDestroyItem[K, V]
	m.mu.Lock()
	entry, ok := m.resources[key]
	if ok {
		delete(m.resources, key)
		destroyItem = entry.markForDestroyLocked()
	}
	m.mu.Unlock()

	m.destroyResource(destroyItem)
	return ok
}

// Sweep removes all timed-out resources that are not currently leased.
func (m *ResourceManager[K, V]) Sweep() {
	var destroyItems []resourceDestroyItem[K, V]

	now := time.Now()
	m.mu.Lock()
	if m.timeout > 0 {
		for key, entry := range m.resources {
			if entry.refCount == 0 && now.Sub(entry.lastAccessTime) >= m.timeout {
				delete(m.resources, key)
				destroyItems = append(destroyItems, entry.markForDestroyLocked())
			}
		}
	}
	m.mu.Unlock()

	for _, item := range destroyItems {
		m.destroyResource(item)
	}
}

// Len returns the number of resources currently reachable from the manager.
func (m *ResourceManager[K, V]) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.resources)
}

// Value returns the leased resource.
func (l *ResourceLease[K, V]) Value() V {
	return l.entry.value
}

// Release releases the lease. It is safe to call Release multiple times.
func (l *ResourceLease[K, V]) Release() {
	l.releaseOnce.Do(func() {
		destroyItem := l.manager.release(l.entry)
		l.manager.destroyResource(destroyItem)
	})
}

func (m *ResourceManager[K, V]) release(entry *resourceEntry[K, V]) resourceDestroyItem[K, V] {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry.refCount > 0 {
		entry.refCount--
	}
	if entry.refCount == 0 && entry.pendingDestroy {
		return entry.markDestroyedLocked()
	}
	return resourceDestroyItem[K, V]{}
}

func (m *ResourceManager[K, V]) sweepLoop() {
	for {
		time.Sleep(m.currentScanInterval())
		m.Sweep()
	}
}

func (m *ResourceManager[K, V]) currentScanInterval() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scanInterval
}

func (m *ResourceManager[K, V]) destroyResource(item resourceDestroyItem[K, V]) {
	if !item.valid || m.destroy == nil {
		return
	}
	m.destroy(item.key, item.value)
}

func (entry *resourceEntry[K, V]) markForDestroyLocked() resourceDestroyItem[K, V] {
	entry.pendingDestroy = true
	if entry.refCount > 0 {
		return resourceDestroyItem[K, V]{}
	}
	return entry.markDestroyedLocked()
}

func (entry *resourceEntry[K, V]) markDestroyedLocked() resourceDestroyItem[K, V] {
	if entry.destroyed {
		return resourceDestroyItem[K, V]{}
	}
	entry.destroyed = true
	return resourceDestroyItem[K, V]{
		key:   entry.key,
		value: entry.value,
		valid: true,
	}
}

type resourceDestroyItem[K comparable, V any] struct {
	key   K
	value V
	valid bool
}

func normalizeScanInterval(timeout time.Duration, scanInterval time.Duration) time.Duration {
	if scanInterval > 0 {
		return scanInterval
	}
	if timeout <= 0 {
		return time.Second
	}
	scanInterval = timeout / 2
	if scanInterval < time.Second {
		return time.Second
	}
	return scanInterval
}
