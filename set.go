package main

type Set struct {
	m map[string]bool
}

func NewSet() *Set {
	return &Set{
		m: make(map[string]bool),
	}
}

func (s *Set) Add(str string) {
	s.m[str] = true
}

func (s *Set) Contains(str string) bool {
	_, ok := s.m[str]
	return ok
}

func (s *Set) Remove(str string) {
	delete(s.m, str)
}
