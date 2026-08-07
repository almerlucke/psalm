package set

type Set[T comparable] map[T]struct{}

func New[T comparable](elems ...T) Set[T] {
	return make(Set[T]).Add(elems...)
}

func (s Set[T]) Add(elems ...T) Set[T] {
	for _, elem := range elems {
		s[elem] = struct{}{}
	}
	return s
}

func (s Set[T]) Remove(elems ...T) Set[T] {
	for _, elem := range elems {
		delete(s, elem)
	}
	return s
}

func (s Set[T]) Contains(elems ...T) bool {
	for _, elem := range elems {
		if _, ok := s[elem]; !ok {
			return false
		}
	}
	return true
}

func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) Empty() bool {
	return len(s) == 0
}

func (s Set[T]) Clear() Set[T] {
	clear(s)
	return s
}

func (s Set[T]) Values() []T {
	values := make([]T, s.Size())
	i := 0
	for v := range s {
		values[i] = v
	}
	return values
}

func (s Set[T]) Copy() Set[T] {
	c := make(Set[T], s.Size())
	for v := range s {
		c[v] = struct{}{}
	}
	return c
}

func (s Set[T]) Equal(other Set[T]) bool {
	return s.Difference(other).Empty()
}

func (s Set[T]) Union(other Set[T]) Set[T] {
	set := make(Set[T])
	for v := range s {
		set[v] = struct{}{}
	}
	for v := range other {
		set[v] = struct{}{}
	}
	return set
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	set := make(Set[T])
	for v := range s {
		if _, ok := other[v]; ok {
			set[v] = struct{}{}
		}
	}
	return set
}

func (s Set[T]) Difference(other Set[T]) Set[T] {
	set := make(Set[T])
	for v := range s {
		if _, ok := other[v]; !ok {
			set[v] = struct{}{}
		}
	}
	for v := range other {
		if _, ok := s[v]; !ok {
			set[v] = struct{}{}
		}
	}
	return set
}
