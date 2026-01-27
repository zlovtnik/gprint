package fp

import (
	"github.com/IBM/fp-go/either"
)

// Result represents a computation that can succeed with a value of type T,
// or fail with an error. This is a type alias for Either[error, T].
type Result[T any] = either.Either[error, T]

// Success creates a successful Result containing the given value.
func Success[T any](value T) Result[T] {
	return either.Right[error](value)
}

// Failure creates a failed Result containing the given error.
func Failure[T any](err error) Result[T] {
	return either.Left[T](err)
}

// IsSuccess checks if the Result is a success.
func IsSuccess[T any](result Result[T]) bool {
	return either.IsRight(result)
}

// IsFailure checks if the Result is a failure.
func IsFailure[T any](result Result[T]) bool {
	return either.IsLeft(result)
}

// GetOrElse returns the value if success, or a default value if failure.
func GetOrElse[T any](defaultValue T) func(Result[T]) T {
	return either.GetOrElse(func(_ error) T { return defaultValue })
}

// GetError extracts the error from a failure, or returns nil if success.
func GetError[T any](result Result[T]) error {
	return either.Fold(
		func(err error) error { return err },
		func(_ T) error { return nil },
	)(result)
}

// GetValue extracts the value from a success, or returns zero value if failure.
func GetValue[T any](result Result[T]) T {
	var zero T
	return either.GetOrElse(func(_ error) T { return zero })(result)
}

// Map applies a function to the value inside a Result.
func Map[A, B any](f func(A) B) func(Result[A]) Result[B] {
	return either.Map[error, A, B](f)
}

// FlatMap chains operations that return Results.
func FlatMap[A, B any](f func(A) Result[B]) func(Result[A]) Result[B] {
	return either.Chain[error, A, B](f)
}

// Fold applies one of two functions based on the Result.
func Fold[T, U any](onFailure func(error) U, onSuccess func(T) U) func(Result[T]) U {
	return either.Fold(onFailure, onSuccess)
}

// MapError transforms the error in a failure.
func MapError[T any](f func(error) error) func(Result[T]) Result[T] {
	return either.MapLeft[T, error, error](f)
}

// FromOption converts an Option to a Result using the provided error for None.
func FromOption[T any](err error) func(Option[T]) Result[T] {
	return func(opt Option[T]) Result[T] {
		return FoldOpt(
			func() Result[T] { return Failure[T](err) },
			func(v T) Result[T] { return Success(v) },
		)(opt)
	}
}

// ToOption converts a Result to an Option, discarding the error.
func ToOption[T any](result Result[T]) Option[T] {
	return Fold(
		func(_ error) Option[T] { return None[T]() },
		func(v T) Option[T] { return Some(v) },
	)(result)
}

// TryCatch wraps a function that may panic and returns a Result.
func TryCatch[T any](f func() T) Result[T] {
	var result Result[T]
	func() {
		defer func() {
			if r := recover(); r != nil {
				switch e := r.(type) {
				case error:
					result = Failure[T](e)
				default:
					result = Failure[T](NewError("panic: %v", e))
				}
			}
		}()
		result = Success(f())
	}()
	return result
}

// Recover attempts to recover from a failure using the given function.
func Recover[T any](f func(error) Result[T]) func(Result[T]) Result[T] {
	return func(result Result[T]) Result[T] {
		return Fold(
			func(err error) Result[T] { return f(err) },
			func(v T) Result[T] { return Success(v) },
		)(result)
	}
}

// Ap applies a function inside a Result to a value inside another Result.
func Ap[A, B any](resultF Result[func(A) B]) func(Result[A]) Result[B] {
	return func(resultA Result[A]) Result[B] {
		return Fold(
			func(err error) Result[B] { return Failure[B](err) },
			func(f func(A) B) Result[B] { return Map(f)(resultA) },
		)(resultF)
	}
}

// Sequence converts a slice of Results into a Result of a slice.
func Sequence[T any](results []Result[T]) Result[[]T] {
	values := make([]T, 0, len(results))
	for _, r := range results {
		if IsFailure(r) {
			return Failure[[]T](GetError(r))
		}
		values = append(values, GetValue(r))
	}
	return Success(values)
}

// Traverse applies a function to each element and collects results.
func Traverse[A, B any](f func(A) Result[B]) func([]A) Result[[]B] {
	return func(items []A) Result[[]B] {
		results := make([]Result[B], len(items))
		for i, item := range items {
			results[i] = f(item)
		}
		return Sequence(results)
	}
}
