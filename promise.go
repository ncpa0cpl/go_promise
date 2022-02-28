package promise

import "sync"

type NULL = interface{}

type Promise[T any] struct {
	mutex      sync.Mutex
	result     T
	err        error
	isFinished bool
	callbacks  []func()
}

func (p *Promise[T]) dispatch(result T, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.result = result
	p.err = err
	p.isFinished = true

	for _, callback := range p.callbacks {
		go callback()
	}

	p.callbacks = nil
}

// Creates a new Promise that will run once this Promise
// goroutine finishes without error.
func (p *Promise[T]) Then(fn func(value T) error) *Promise[T] {
	return New(func() (T, error) {
		r, parentError := p.Await()

		if parentError == nil {
			thenError := fn(r)
			return r, thenError
		}

		return r, parentError
	})
}

// Creates a new Promise that will run once this Promise
// goroutine finishes with error.
func (p *Promise[T]) Catch(fn func(err error) error) *Promise[NULL] {
	return New(func() (NULL, error) {
		_, parentError := p.Await()

		if parentError != nil {
			catchError := fn(parentError)
			return nil, catchError
		}

		return nil, parentError
	})
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
	p.mutex.Lock()
	if p.isFinished {
		return p.result, p.err
	}

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	p.callbacks = append(p.callbacks, func() {
		waitGroup.Done()
	})
	p.mutex.Unlock()

	waitGroup.Wait()

	return p.result, p.err
}

func (p *Promise[T]) IsPending() bool {
	return !p.isFinished
}
