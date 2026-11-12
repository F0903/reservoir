package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
)

// A flight is one shared fetch. Every request for the same key that arrives
// while it runs waits on it instead of asking upstream again.
type flight struct {
	done   chan struct{} // Closed once result and err are set.
	result Result
	err    error

	started    chan struct{} // Closed once body is set.
	body       *liveBody     // The response body, while it is being downloaded into the cache.
	bodyStatus Status        // How a response read from body is reported.

	callers int // Guarded by flightGroup.mu.
}

// startBody lets the waiting requests read the body while it is still downloading.
func (fl *flight) startBody(body *liveBody) {
	fl.body = body
	close(fl.started)
}

// liveBody returns the body being downloaded, or nil if the flight has not got one.
func (fl *flight) liveBody() *liveBody {
	select {
	case <-fl.started:
		return fl.body
	default:
		return nil
	}
}

type flightGroup struct {
	mu      sync.Mutex
	flights map[string]*flight
}

// join returns the running flight for key, or starts one that runs fn.
func (g *flightGroup) join(key string, fn func(*flight) (Result, error)) *flight {
	g.mu.Lock()
	defer g.mu.Unlock()

	fl, running := g.flights[key]
	if !running {
		fl = &flight{done: make(chan struct{}), started: make(chan struct{}), bodyStatus: StatusMiss}
		if g.flights == nil {
			g.flights = make(map[string]*flight)
		}
		g.flights[key] = fl
		go g.run(key, fl, fn)
	}
	fl.callers++
	return fl
}

func (g *flightGroup) run(key string, fl *flight, fn func(*flight) (Result, error)) {
	defer func() {
		// The flight has its own goroutine, so a panic here would take the whole proxy down.
		if r := recover(); r != nil {
			slog.Error("Panic in shared fetch", "panic", r, "stack", string(debug.Stack()))
			fl.err = fmt.Errorf("panic in shared fetch: %v", r)
		}

		g.mu.Lock()
		delete(g.flights, key)
		g.mu.Unlock()

		// Only now: a request that follows a finished body must start a new flight
		// and find the cache entry, not join the end of this one.
		if fl.body != nil {
			fl.body.finish()
		}
		close(fl.done)
	}()

	fl.result, fl.err = fn(fl)
}

// shared reports whether more than one request has joined the flight so far.
func (g *flightGroup) shared(fl *flight) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	return fl.callers > 1
}

type flightContextKey struct{}

func withFlight(ctx context.Context, fl *flight) context.Context {
	return context.WithValue(ctx, flightContextKey{}, fl)
}

// flightFrom returns the flight a request is fetched for, or nil if nobody else can be waiting on it.
func flightFrom(ctx context.Context) *flight {
	fl, _ := ctx.Value(flightContextKey{}).(*flight)
	return fl
}
