package mapmutex

import (
	"context"
	"fmt"
	"sync"

	"github.com/swmh/gopetbin/internal/service"
)

type MutexEntry[T comparable] struct {
	origMap *MutexMap[T]
	mu      sync.Mutex
	cnt     int
	key     T
}

type MutexMap[T comparable] struct {
	mu sync.Mutex
	m  map[T]*MutexEntry[T]
}

func New[T comparable]() *MutexMap[T] {
	return &MutexMap[T]{
		mu: sync.Mutex{},
		m:  make(map[T]*MutexEntry[T]),
	}
}

func (me *MutexEntry[T]) Unlock(_ context.Context) error {
	origMap := me.origMap

	origMap.mu.Lock()

	elMutex, ok := origMap.m[me.key]
	if !ok {
		origMap.mu.Unlock()
		return fmt.Errorf("cannot unlock key=%v, no entry found", me.key)
	}

	elMutex.cnt--
	if elMutex.cnt < 1 {
		delete(origMap.m, me.key)
	}

	origMap.mu.Unlock()
	elMutex.mu.Unlock()

	return nil
}

func (m *MutexMap[T]) Lock(_ context.Context, id T) (service.Mutex, error) {
	return m.lock(id), nil
}

func (m *MutexMap[T]) lock(id T) *MutexEntry[T] {
	m.mu.Lock()

	elMutex, ok := m.m[id]
	if !ok {
		elMutex = &MutexEntry[T]{
			origMap: m,
			key:     id,
		}
		m.m[id] = elMutex
	}

	elMutex.cnt++
	m.mu.Unlock()

	elMutex.mu.Lock()
	return elMutex
}
