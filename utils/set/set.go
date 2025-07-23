package set

import (
	"iter"
	"maps"
)

// Set is a collection of unique elements
type Set[T comparable] struct {
	elements map[T]struct{}
}

func New[T comparable]() *Set[T] {
	return &Set[T]{
		elements: make(map[T]struct{}),
	}
}

func (s *Set[T]) Add(value T) {
	s.elements[value] = struct{}{}
}

func (s *Set[T]) Remove(value T) {
	delete(s.elements, value)
}

func (s *Set[T]) Contains(value T) bool {
	_, found := s.elements[value]
	return found
}

func (s *Set[T]) Size() int {
	return len(s.elements)
}

func (s *Set[T]) Iter() iter.Seq[T] {
	return maps.Keys(s.elements)
}
