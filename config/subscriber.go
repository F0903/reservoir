package config

type ConfigSubscriber struct {
	unsubs []func()
}

func (s *ConfigSubscriber) Add(unsub func()) {
	s.unsubs = append(s.unsubs, unsub)
}

func (s *ConfigSubscriber) UnsubscribeAll() {
	for _, unsub := range s.unsubs {
		if unsub != nil {
			unsub()
		}
	}
	s.unsubs = nil
}
