package routes

import "net/http"

type MetricsRoute struct{}

func (m *MetricsRoute) Name() string {
	return "metrics"
}

func (m *MetricsRoute) HandleRoute(path string, w http.ResponseWriter, r *http.Request) error {
	_, err := w.Write([]byte("Hello from MetricsRoute!"))
	return err
}
