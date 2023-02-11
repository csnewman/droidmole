package broadcaster

import (
	"errors"
	"sync"
)

var Closed = errors.New("broadcaster closed")

type Broadcaster[T interface{}] struct {
	mu         *sync.Mutex
	cond       *sync.Cond
	generation uint64
	value      T
	closed     bool
}

type Listener[T interface{}] struct {
	broadcaster    *Broadcaster[T]
	lastGeneration uint64
}

func New[T interface{}]() *Broadcaster[T] {
	mu := &sync.Mutex{}

	return &Broadcaster[T]{
		mu:   mu,
		cond: sync.NewCond(mu),
	}
}

func (b *Broadcaster[T]) Broadcast(value T) {
	b.mu.Lock()
	b.generation++
	b.value = value
	b.cond.Broadcast()
	b.mu.Unlock()
}

func (b *Broadcaster[T]) Close() {
	b.mu.Lock()
	b.closed = true
	b.cond.Broadcast()
	b.mu.Unlock()
}

func (b *Broadcaster[T]) Listener() *Listener[T] {
	return &Listener[T]{
		broadcaster: b,
	}
}

func (l *Listener[T]) Wait() (T, error) {
	var result T
	var err error

	l.broadcaster.mu.Lock()

	if l.broadcaster.closed {
		err = Closed
	} else {
		if l.broadcaster.generation <= l.lastGeneration {
			l.broadcaster.cond.Wait()
		}

		if l.broadcaster.generation > l.lastGeneration {
			l.lastGeneration = l.broadcaster.generation
			result = l.broadcaster.value
		} else if l.broadcaster.closed {
			err = Closed
		} else {
			panic("broadcaster has inconsistent state")
		}
	}

	l.broadcaster.mu.Unlock()

	return result, err
}
