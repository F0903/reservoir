package event

import "sync"

type EventFn[T any] func(data T)

type Unsubscribe func()

var initMu sync.Mutex

type Event[T any] struct {
	state *eventState[T]
}

type eventState[T any] struct {
	mu          sync.RWMutex
	nextID      uint64
	order       []uint64
	subscribers map[uint64]EventFn[T]
}

func New[T any]() *Event[T] {
	event := NewEvent[T]()
	return &event
}

func NewEvent[T any]() Event[T] {
	return Event[T]{
		state: newEventState[T](),
	}
}

func newEventState[T any]() *eventState[T] {
	return &eventState[T]{
		subscribers: make(map[uint64]EventFn[T]),
	}
}

func (e *Event[T]) ensureState() *eventState[T] {
	initMu.Lock()
	defer initMu.Unlock()
	if e.state == nil {
		e.state = newEventState[T]()
	}
	return e.state
}

// Adds a subscriber to the event.
func (e *Event[T]) Subscribe(fn EventFn[T]) Unsubscribe {
	if fn == nil {
		return func() {}
	}

	state := e.ensureState()
	state.mu.Lock()
	id := state.nextID
	state.nextID++
	state.subscribers[id] = fn
	state.order = append(state.order, id)
	state.mu.Unlock()

	return func() {
		state.mu.Lock()
		defer state.mu.Unlock()

		if _, ok := state.subscribers[id]; !ok {
			return
		}
		delete(state.subscribers, id)
		for i, existingID := range state.order {
			if existingID == id {
				state.order = append(state.order[:i], state.order[i+1:]...)
				break
			}
		}
	}
}

// Fires the event, notifying all subscribers with the provided data.
func (e *Event[T]) Fire(data T) {
	state := e.ensureState()

	state.mu.RLock()
	subscribers := make([]EventFn[T], 0, len(state.subscribers))
	for _, id := range state.order {
		if subscriber, ok := state.subscribers[id]; ok {
			subscribers = append(subscribers, subscriber)
		}
	}
	state.mu.RUnlock()

	for _, subscriber := range subscribers {
		subscriber(data)
	}
}
