package promise

import "sync"

type awaiterResult[T any] struct {
	value T
	err   error
}

type Promise[T any] struct {
	mutex      sync.Mutex
	result     T
	err        error
	isFinished bool
	awaiters   []chan awaiterResult[T]
}

func (p *Promise[T]) finalize(result T, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.result = result
	p.err = err
	p.isFinished = true

	for _, channel := range p.awaiters {
		channel <- awaiterResult[T]{value: result, err: err}
	}

	p.awaiters = []chan awaiterResult[T]{}
}

// Creates a new Promise that will run once this Promise
// goroutine finishes.
func (p *Promise[T]) Finally(fn func(value T, err error) error) *Promise[T] {
	return New(func() (T, error) {
		r, parentError := p.Await()

		finallyError := fn(r, parentError)
		return r, finallyError
	})
}

// Waits for this Promise goroutine to finish and returns it's result.
func (p *Promise[T]) Await() (T, error) {
	if p.isFinished {
		return p.result, p.err
	}

	p.mutex.Lock()

	if p.isFinished {
		p.mutex.Unlock()
		return p.result, p.err
	}

	channel := make(chan awaiterResult[T])

	p.awaiters = append(p.awaiters, channel)

	p.mutex.Unlock()

	result := <-channel

	return result.value, result.err
}

func (p *Promise[T]) IsPending() bool {
	return !p.isFinished
}

func (p *Promise[T]) Read() (T, bool) {
	if p.isFinished {
		return p.result, true
	}

	var t T
	return t, false
}
