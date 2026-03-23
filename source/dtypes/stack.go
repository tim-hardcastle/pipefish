package dtypes

type Stack[T comparable] struct {
	vals []T
}

func NewStack[T comparable]() *Stack[T] { return &Stack[T]{vals: []T{}} }

func (s *Stack[T]) Len() int {
	return len(s.vals)
}

func (s *Stack[T]) Copy() *Stack[T] {
	new := NewStack[T]()
	new.vals = make([]T, len(s.vals))
	copy(new.vals, s.vals)
	return new
}

func (s *Stack[T]) Push(val T) {
	s.vals = append(s.vals, val)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.vals) == 0 {
		var zero T
		return zero, false
	}
	top := s.vals[len(s.vals)-1]
	s.vals = s.vals[:len(s.vals)-1]
	return top, true
}

func (s *Stack[T]) Take(n int) ([]T, bool) {
	lb := len(s.vals) - n
	if lb < 0 {
		return nil, false
	}
	result := s.vals[lb:]
	s.vals = s.vals[:lb]
	return result, true
}

func (s *Stack[T]) HeadValue() (T, bool) {
	if len(s.vals) == 0 {
		var zero T
		return zero, false
	}
	top := s.vals[len(s.vals)-1]
	return top, true
}

func (S Stack[T]) Find(e T) int {
	level := -1
	for i := len(S.vals) - 1; i >= 0; i-- {
		level++
		if S.vals[i] == e {
			return level
		}
	}
	return -1
}
