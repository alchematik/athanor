package set

import (
	"sync"
)

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		values: map[T]struct{}{},
	}
}

type Set[T comparable] struct {
	sync.Mutex

	values map[T]struct{}
}

func (s *Set[T]) Add(val T) {
	s.Lock()
	defer s.Unlock()

	s.values[val] = struct{}{}
}

func (s *Set[T]) Remove(val T) {
	s.Lock()
	defer s.Unlock()

	delete(s.values, val)
}

func (s *Set[T]) Clone() *Set[T] {
	s.Lock()
	defer s.Unlock()

	dest := NewSet[T]()
	for k := range s.values {
		dest.Add(k)
	}

	return dest
}

func (s *Set[T]) Len() int {
	s.Lock()
	defer s.Unlock()

	return len(s.values)
}

func (s *Set[T]) Values() []T {
	s.Lock()
	defer s.Unlock()

	vals := make([]T, 0, len(s.values))
	for v := range s.values {
		vals = append(vals, v)
	}

	return vals
}
