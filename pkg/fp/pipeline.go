package fp

import (
	"errors"
	"fmt"
)

// Pipeline provides a fluent API for chaining Result operations.
type Pipeline[T any] struct {
	result Result[T]
}

// NewPipeline creates a new Pipeline from an initial value.
func NewPipeline[T any](value T) *Pipeline[T] {
	return &Pipeline[T]{result: Success(value)}
}

// FromResult creates a Pipeline from an existing Result.
func FromResult[T any](result Result[T]) *Pipeline[T] {
	return &Pipeline[T]{result: result}
}

// FromError creates a failed Pipeline.
func FromError[T any](err error) *Pipeline[T] {
	return &Pipeline[T]{result: Failure[T](err)}
}

// Map applies a transformation to the value.
func (p *Pipeline[T]) Map(f func(T) T) *Pipeline[T] {
	p.result = Map(f)(p.result)
	return p
}

// FlatMap chains with another operation that returns a Result.
func (p *Pipeline[T]) FlatMap(f func(T) Result[T]) *Pipeline[T] {
	p.result = FlatMap(f)(p.result)
	return p
}

// Filter applies a predicate and fails if it returns false.
func (p *Pipeline[T]) Filter(predicate func(T) bool, errMsg string) *Pipeline[T] {
	p.result = FlatMap(func(v T) Result[T] {
		if predicate(v) {
			return Success(v)
		}
		return Failure[T](fmt.Errorf("%s", errMsg))
	})(p.result)
	return p
}

// Tap executes a side effect without modifying the value.
func (p *Pipeline[T]) Tap(f func(T)) *Pipeline[T] {
	p.result = Map(func(v T) T {
		f(v)
		return v
	})(p.result)
	return p
}

// MapError transforms the error if the pipeline failed.
func (p *Pipeline[T]) MapError(f func(error) error) *Pipeline[T] {
	p.result = MapError[T](f)(p.result)
	return p
}

// Recover attempts to recover from a failure.
func (p *Pipeline[T]) Recover(f func(error) Result[T]) *Pipeline[T] {
	p.result = Recover(f)(p.result)
	return p
}

// Result returns the final Result.
func (p *Pipeline[T]) Result() Result[T] {
	return p.result
}

// Unwrap returns the value or panics if failed.
func (p *Pipeline[T]) Unwrap() T {
	return Fold(
		func(err error) T { panic(err) },
		func(v T) T { return v },
	)(p.result)
}

// UnwrapOr returns the value or a default if failed.
func (p *Pipeline[T]) UnwrapOr(defaultValue T) T {
	return GetOrElse(defaultValue)(p.result)
}

// Do executes a function and wraps it in a Pipeline.
func Do[T any](f func() (T, error)) *Pipeline[T] {
	v, err := f()
	if err != nil {
		return FromError[T](err)
	}
	return NewPipeline(v)
}

// Then chains a transformation that returns a new type.
func Then[A, B any](p *Pipeline[A], f func(A) Result[B]) *Pipeline[B] {
	return FromResult(FlatMap(f)(p.result))
}

// MapTo transforms the pipeline value to a new type.
func MapTo[A, B any](p *Pipeline[A], f func(A) B) *Pipeline[B] {
	return FromResult(Map(f)(p.result))
}

// Bind is an alias for Then for more functional style.
func Bind[A, B any](f func(A) Result[B]) func(*Pipeline[A]) *Pipeline[B] {
	return func(p *Pipeline[A]) *Pipeline[B] {
		return Then(p, f)
	}
}

// Compose combines multiple transformations.
func Compose[A, B, C any](f func(A) Result[B], g func(B) Result[C]) func(A) Result[C] {
	return func(a A) Result[C] {
		return FlatMap(g)(f(a))
	}
}

// Pipe applies a sequence of transformations.
func Pipe[T any](value T, transforms ...func(T) Result[T]) Result[T] {
	result := Success(value)
	for _, t := range transforms {
		if IsFailure(result) {
			return result
		}
		result = FlatMap(t)(result)
	}
	return result
}

// All returns Success if all results are successful, otherwise the first failure.
func All[T any](results ...Result[T]) Result[[]T] {
	return Sequence(results)
}

// Any returns the first success, or the last failure if all fail.
func Any[T any](results ...Result[T]) Result[T] {
	if len(results) == 0 {
		return Failure[T](errors.New("no results provided"))
	}
	var lastErr error
	for _, r := range results {
		if IsSuccess(r) {
			return r
		}
		lastErr = GetError(r)
	}
	return Failure[T](lastErr)
}
