package event

import (
	"slices"
	"sync"
	"testing"
)

func TestUnsubscribeOutOfOrder(t *testing.T) {
	event := New[int]()
	var calls []int

	unsubOne := event.Subscribe(func(int) {
		calls = append(calls, 1)
	})
	unsubTwo := event.Subscribe(func(int) {
		calls = append(calls, 2)
	})
	unsubThree := event.Subscribe(func(int) {
		calls = append(calls, 3)
	})

	unsubOne()
	unsubThree()
	unsubOne()
	unsubThree()

	event.Fire(0)

	if !slices.Equal(calls, []int{2}) {
		t.Fatalf("expected only the second subscriber to fire, got %v", calls)
	}

	unsubTwo()
	event.Fire(0)

	if !slices.Equal(calls, []int{2}) {
		t.Fatalf("expected no subscribers after unsubscribe, got %v", calls)
	}
}

func TestFireUsesSubscriptionOrder(t *testing.T) {
	event := New[int]()
	var calls []int

	event.Subscribe(func(int) {
		calls = append(calls, 1)
	})
	event.Subscribe(func(int) {
		calls = append(calls, 2)
	})
	event.Subscribe(func(int) {
		calls = append(calls, 3)
	})

	event.Fire(0)

	if !slices.Equal(calls, []int{1, 2, 3}) {
		t.Fatalf("expected subscribers to fire in subscription order, got %v", calls)
	}
}

func TestConcurrentSubscribeFireUnsubscribe(t *testing.T) {
	event := New[int]()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			unsub := event.Subscribe(func(int) {})
			event.Fire(0)
			unsub()
			unsub()
		}()
	}

	wg.Wait()
}

func TestZeroValueEventConcurrentUse(t *testing.T) {
	var event Event[int]
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			unsub := event.Subscribe(func(int) {})
			event.Fire(0)
			unsub()
		}()
	}

	wg.Wait()
}
