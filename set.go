package main

import (
	"sync"
)

type Set struct {
	m     map[string]bool
	mutex *sync.RWMutex
}

func NewSet() *Set {
	return &Set{
		m:     make(map[string]bool),
		mutex: &sync.RWMutex{},
	}
}

func (s *Set) Add(str string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.m[str] = true
}

func (s *Set) Contains(str string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok := s.m[str]
	return ok
}

func (s *Set) Iterate(f func(str string)) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for k, _ := range s.m {
		f(k)
	}
}

func (s *Set) Remove(str string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.m, str)
}
