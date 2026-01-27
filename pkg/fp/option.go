package fp

import (
	"github.com/IBM/fp-go/option"
)

// Option represents an optional value (Some or None).
type Option[T any] = option.Option[T]

// Some wraps a value in an Option.
func Some[T any](value T) Option[T] {
	return option.Some(value)
}

// None returns an empty Option.
func None[T any]() Option[T] {
	return option.None[T]()
}

// FromPointer converts a pointer to an Option.
// Returns None if the pointer is nil, otherwise returns Some with the dereferenced value.
func FromPointer[T any](ptr *T) Option[T] {
	if ptr == nil {
		return option.None[T]()
	}
	return option.Some(*ptr)
}

// ToPointer converts an Option to a pointer.
// Returns nil if None, otherwise returns a pointer to the value.
func ToPointer[T any](opt Option[T]) *T {
	return option.Fold(
		func() *T { return nil },
		func(v T) *T { return &v },
	)(opt)
}

// IsSome checks if an Option contains a value.
func IsSome[T any](opt Option[T]) bool {
	return option.IsSome(opt)
}

// IsNone checks if an Option is empty.
func IsNone[T any](opt Option[T]) bool {
	return option.IsNone(opt)
}

// GetOrElseOpt returns the value if Some, or a default value if None.
func GetOrElseOpt[T any](defaultValue T) func(Option[T]) T {
	return option.GetOrElse(func() T { return defaultValue })
}

// MapOpt applies a function to the value inside an Option.
func MapOpt[A, B any](f func(A) B) func(Option[A]) Option[B] {
	return option.Map[A, B](f)
}

// FlatMapOpt chains operations that return Options.
func FlatMapOpt[A, B any](f func(A) Option[B]) func(Option[A]) Option[B] {
	return option.Chain[A, B](f)
}

// FoldOpt applies one of two functions based on the Option.
func FoldOpt[T, U any](onNone func() U, onSome func(T) U) func(Option[T]) U {
	return option.Fold(onNone, onSome)
}

// Filter returns None if the predicate returns false.
func Filter[T any](predicate func(T) bool) func(Option[T]) Option[T] {
	return option.Filter(predicate)
}
