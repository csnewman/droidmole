package testutil

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func RunParallel(t *testing.T, funcs ...func(*testing.T)) error {
	wg := &sync.WaitGroup{}
	wg.Add(len(funcs))

	for _, f := range funcs {
		fCopy := f
		go func() {
			t.Run("parallel", fCopy)
			wg.Done()
		}()
	}

	select {
	case <-wrapWait(wg):
		return nil
	case <-time.NewTimer(500 * time.Millisecond).C:
		return errors.New("test parallel timeout")
	}
}

func wrapWait(wg *sync.WaitGroup) <-chan struct{} {
	out := make(chan struct{})
	go func() {
		wg.Wait()
		out <- struct{}{}
	}()
	return out
}
