package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/proxy/responder"
)

func finalizeAndRespond(m *metrics.Metrics, r responder.Responder, resp io.Reader, status int, req *http.Request) error {
	m = metrics.Ensure(m)
	body := resp
	if req.Method == http.MethodHead {
		body = http.NoBody
	}

	switch {
	case status >= 200 && status < 300:
		m.Requests.StatusOKResponses.Increment()
	case status >= 400 && status < 500:
		m.Requests.StatusClientErrorResponses.Increment()
	case status >= 500 && status < 600:
		m.Requests.StatusServerErrorResponses.Increment()
	}

	written, writeDuration, err := r.Write(status, body)
	m.Requests.ClientResponseLatency.Add(writeDuration.Nanoseconds())
	m.Requests.ClientResponses.Increment()
	if err != nil {
		slog.Error("Error writing response", "url", req.URL, "error", err)
		return err
	}

	m.Requests.BytesServed.Add(written)
	slog.Info("Response sent", "url", req.URL, "bytes_written", written)
	return nil
}
