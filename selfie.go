package psalm

type Selfie[T any] interface {
	Self() T
	SetSelf(T)
}
