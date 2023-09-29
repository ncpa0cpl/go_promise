package go_promise

import (
	"sync"
)

type PromiseList[T any] []*Promise[T]

type AwaitAllResult[T any] struct {
	Result T
	Err    error
}

func runGoRoutine[T any](fn func() (T, error), onEnd func(result T, err error)) {
	result, err := fn()
	onEnd(result, err)
}

// Creates new goroutine wrapped in a Promise struct. Promise struct exposes methods for, waiting
// for the routine to finish, or attaching listener functions.
func New[T any](fn func() (T, error)) *Promise[T] {
	p := Promise[T]{}

	go runGoRoutine(fn, func(result T, err error) {
		p.dispatch(result, err)
	})

	return &p
}

// Creates a new goroutine which waits for the provided Promise to resolve, and runs the provided
// function with the result of the Promise as it's arguments.
func Pipe[T any, U any](promise *Promise[T], fn func(value T, err error) (U, error)) *Promise[U] {
	return New(func() (U, error) {
		r, e := promise.Await()
		return fn(r, e)
	})
}

// Waits for all of the Promises in the provided list to resolve, and returns a list
// with each of those Promises result.
func AwaitAll[T any](promises PromiseList[T]) *[]AwaitAllResult[T] {
	results := make([]AwaitAllResult[T], len(promises))

	var waitGroup sync.WaitGroup

	for i, p := range promises {
		i := i
		waitGroup.Add(1)
		p.Finally(func(value T, err error) error {
			results[i] = AwaitAllResult[T]{Result: value, Err: err}
			waitGroup.Done()

			return nil
		})
	}

	waitGroup.Wait()

	return &results
}

// Waits for any of the provided Promises to resolve, and returns it's result.
func Race[T any](promises PromiseList[T]) (T, error) {
	var result AwaitAllResult[T]
	done := false

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)

	for _, p := range promises {
		p.Finally(func(value T, err error) error {
			if done {
				return err
			}
			done = true

			result.Result = value
			result.Err = err

			waitGroup.Done()

			return nil
		})
	}

	waitGroup.Wait()

	return result.Result, result.Err
}
