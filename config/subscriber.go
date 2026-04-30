package config

import "sync"

type ConfigSubscriber struct {
	mu     sync.Mutex
	unsubs []func()
	closed bool
}

func (s *ConfigSubscriber) Add(unsub func()) {
	if unsub == nil {
		return
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		unsub()
		return
	}
	s.unsubs = append(s.unsubs, unsub)
	s.mu.Unlock()
}

func (s *ConfigSubscriber) UnsubscribeAll() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	unsubs := s.unsubs
	s.unsubs = nil
	s.mu.Unlock()

	for _, unsub := range unsubs {
		if unsub != nil {
			unsub()
		}
	}
}
