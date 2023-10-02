package promise

import "sync"

type promiseList[T any] []*Promise[T]

type AwaitAllSettledResult[T any] struct {
	Result T
	Err    error
}

// Creates new goroutine wrapped in a Promise struct. Promise struct exposes methods for, waiting
// for the routine to finish, or attaching listener functions.
func New[T any](fn func() (T, error)) *Promise[T] {
	p := Promise[T]{
		mutex:      sync.Mutex{},
		isFinished: false,
		awaiters:   []chan awaiterResult[T]{},
		err:        nil,
	}

	go (func() {
		result, err := fn()
		p.finalize(result, err)
	})()

	return &p
}

// Creates a new Promise that will run once this Promise
// goroutine finishes without error.
func Then[T any, U any](p *Promise[T], fn func(value T) (U, error)) *Promise[U] {
	return New(func() (U, error) {
		r, parentError := p.Await()

		if parentError == nil {
			return fn(r)
		}

		var u U
		return u, parentError
	})
}

// Creates a new Promise that will run once this Promise
// goroutine finishes with error.
func Catch[T any, U any](p *Promise[T], fn func(err error) (U, error)) *Promise[U] {
	return New(func() (U, error) {
		_, parentError := p.Await()

		if parentError != nil {
			return fn(parentError)
		}

		var u U
		return u, parentError
	})
}

// Waits for all of the Promises in the provided list to resolve, and returns a list
// with each of those Promises result.
func AwaitAll[T any](promises promiseList[T]) ([]T, []error) {
	results := make([]T, len(promises))
	var errors []error

	var waitGroup sync.WaitGroup

	for i, p := range promises {
		idx := i
		waitGroup.Add(1)

		Then(p, func(value T) (T, error) {
			results[idx] = value
			waitGroup.Done()
			return value, nil
		})

		Catch(p, func(e error) (error, error) {
			waitGroup.Done()
			errors = append(errors, e)
			return e, nil
		})
	}

	waitGroup.Wait()

	return results, errors
}

// Waits for all of the Promises in the provided list to resolve, and returns a list
// with each of those Promises result.
func AwaitAllSettled[T any](promises promiseList[T]) []AwaitAllSettledResult[T] {
	results := make([]AwaitAllSettledResult[T], len(promises))

	var waitGroup sync.WaitGroup

	for i, p := range promises {
		idx := i
		waitGroup.Add(1)

		p.Finally(func(value T, err error) error {
			results[idx] = AwaitAllSettledResult[T]{Result: value, Err: err}
			waitGroup.Done()
			return nil
		})
	}

	waitGroup.Wait()

	return results
}

// Waits for any of the provided Promises to resolve, and returns it's result.
func Race[T any](promises promiseList[T]) (T, error) {
	channel := make(chan T)
	var err error = nil

	failedCount := 0

	for _, p := range promises {
		Then(p, func(value T) (T, error) {
			channel <- value
			return value, nil
		})

		Catch(p, func(e error) (error, error) {
			failedCount++
			if failedCount == len(promises) {
				err = e
				var t T
				channel <- t
			}
			return e, nil
		})
	}

	return <-channel, err
}
