package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/proxy/responder"
	"time"
)

func recordResponseStatus(m *metrics.Metrics, status int) {
	switch {
	case status >= 200 && status < 300:
		m.Requests.StatusOKResponses.Increment()
	case status >= 400 && status < 500:
		m.Requests.StatusClientErrorResponses.Increment()
	case status >= 500 && status < 600:
		m.Requests.StatusServerErrorResponses.Increment()
	}
}

// writeError sends an error the proxy generated itself and counts it like any other response.
func (p *Proxy) writeError(r responder.Responder, message string, status int) {
	recordResponseStatus(p.metrics, status)
	p.metrics.Requests.ClientResponses.Increment()

	start := time.Now()
	r.WriteError(message, status)
	p.metrics.Requests.ClientResponseLatency.Add(time.Since(start).Nanoseconds())
}

func finalizeAndRespond(m *metrics.Metrics, r responder.Responder, resp io.Reader, status int, req *http.Request) error {
	m = metrics.Ensure(m)
	body := resp
	if req.Method == http.MethodHead {
		body = http.NoBody
	}

	recordResponseStatus(m, status)

	written, writeDuration, err := r.Write(status, body)
	m.Requests.ClientResponseLatency.Add(writeDuration.Nanoseconds())
	m.Requests.ClientResponses.Increment()
	m.Requests.BytesServed.Add(written)
	if err != nil {
		slog.Error("Error writing response", "url", req.URL, "error", err)
		return err
	}

	slog.Info("Response sent", "url", req.URL, "bytes_written", written)
	return nil
}
