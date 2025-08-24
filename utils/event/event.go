package event

type EventFn[T any] func(data T)

type Unsubscribe func()

type Event[T any] struct {
	subscribers []EventFn[T]
}

func New[T any]() *Event[T] {
	return &Event[T]{}
}

func (e *Event[T]) Subscribe(fn EventFn[T]) Unsubscribe {
	index := len(e.subscribers)
	e.subscribers = append(e.subscribers, fn)
	return func() {
		e.subscribers = append(e.subscribers[:index], e.subscribers[index+1:]...)
	}
}

func (e *Event[T]) Fire(data T) {
	for _, subscriber := range e.subscribers {
		subscriber(data)
	}
}
