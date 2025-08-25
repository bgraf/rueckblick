package option

type Option[T any] struct {
	value  T
	isSome bool
}

func None[T any]() Option[T] {
	return Option[T]{}
}

func Some[T any](value T) Option[T] {
	return Option[T]{value: value, isSome: true}
}

func (x *Option[T]) IsSome() bool {
	return x.isSome
}

func (x *Option[T]) IsNone() bool {
	return !x.isSome
}

func (x *Option[T]) Get() T {
	if !x.isSome {
		panic("option is none")
	}
	return x.value
}
