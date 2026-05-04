package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/proxy/responder"
)

func finalizeAndRespond(r responder.Responder, resp io.Reader, status int, req *http.Request) error {
	body := resp
	if req.Method == http.MethodHead {
		body = http.NoBody
	}

	switch {
	case status >= 200 && status < 300:
		metrics.Global.Requests.StatusOKResponses.Increment()
	case status >= 400 && status < 500:
		metrics.Global.Requests.StatusClientErrorResponses.Increment()
	case status >= 500 && status < 600:
		metrics.Global.Requests.StatusServerErrorResponses.Increment()
	}

	written, writeDuration, err := r.Write(status, body)
	metrics.Global.Requests.ClientResponseLatency.Add(writeDuration.Nanoseconds())
	metrics.Global.Requests.ClientResponses.Increment()
	if err != nil {
		slog.Error("Error writing response", "url", req.URL, "error", err)
		return err
	}

	metrics.Global.Requests.BytesServed.Add(written)
	slog.Info("Response sent", "url", req.URL, "bytes_written", written)
	return nil
}
