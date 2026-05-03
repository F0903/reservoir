package log

import "testing"

func TestLogEndpointsAllowAuthenticatedReaders(t *testing.T) {
	endpoints := map[string]bool{
		"log":    (&LogEndpoint{}).EndpointMethods()[0].RequiresAdmin,
		"stream": (&LogStreamEndpoint{}).EndpointMethods()[0].RequiresAdmin,
	}

	for name, requiresAdmin := range endpoints {
		t.Run(name, func(t *testing.T) {
			if requiresAdmin {
				t.Fatal("expected log endpoint to allow non-admin authenticated users")
			}
		})
	}
}
