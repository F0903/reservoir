package streaming

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"time"
)

type SseStreamer[Store any] interface {
	Tick(w http.ResponseWriter, writeStream func([]byte) error, store *Store) error
}

type SseStream[Store any] struct {
	header         http.Header
	writer         http.ResponseWriter
	flusher        http.Flusher
	heartbeat      *time.Ticker
	ticker         *time.Ticker
	requestContext context.Context
	streamer       SseStreamer[Store]
	streamerStore  Store
}

func NewSseStream[Store any](header http.Header, writer http.ResponseWriter, flusher http.Flusher, heartbeatInterval time.Duration, tickInterval time.Duration, requestContext context.Context, streamer SseStreamer[Store], streamerStore Store) *SseStream[Store] {
	me := &SseStream[Store]{
		header:         header,
		writer:         writer,
		flusher:        flusher,
		heartbeat:      time.NewTicker(heartbeatInterval),
		ticker:         time.NewTicker(tickInterval),
		requestContext: requestContext,
		streamer:       streamer,
		streamerStore:  streamerStore,
	}

	return me
}

func (s *SseStream[Store]) writeStream(line []byte) error {
	// SSE frame: data:<line>\n\n

	// Avoid CR in Windows \r\n
	line = bytes.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return nil
	}

	if _, err := s.writer.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := s.writer.Write(line); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte("\n\n")); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

func (s *SseStream[Store]) Start() error {
	s.header.Set("Content-Type", "text/event-stream")
	s.header.Set("Cache-Control", "no-cache")
	s.header.Set("Connection", "keep-alive")

	slog.Debug("starting SSE stream")

	for {
		select {
		case <-s.requestContext.Done():
			slog.Debug("SSE stream done")
			return nil

		case <-s.heartbeat.C:
			// SSE heartbeat (comment format)
			_, _ = s.writer.Write([]byte(": ping\n\n"))
			s.flusher.Flush()
			slog.Debug("sent SSE stream heartbeat")

		case <-s.ticker.C:
			slog.Debug("running SSE tick")
			if err := s.streamer.Tick(s.writer, s.writeStream, &s.streamerStore); err != nil {
				slog.Error("SSE tick failed", "error", err)
				return err
			}
		}
	}
}

func (s *SseStream[Store]) Close() {
	s.heartbeat.Stop()
	s.ticker.Stop()
}
