package factory

type Factory[T any] interface {
	New(any) T
}
