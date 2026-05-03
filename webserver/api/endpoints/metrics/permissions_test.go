package metrics

import "testing"

func TestMetricsEndpointsAllowAuthenticatedReaders(t *testing.T) {
	endpoints := map[string]bool{
		"all":      (&AllMetricsEndpoint{}).EndpointMethods()[0].RequiresAdmin,
		"cache":    (&CacheMetricsEndpoint{}).EndpointMethods()[0].RequiresAdmin,
		"requests": (&RequestsMetricsEndpoint{}).EndpointMethods()[0].RequiresAdmin,
		"system":   (&SystemMetricsEndpoint{}).EndpointMethods()[0].RequiresAdmin,
	}

	for name, requiresAdmin := range endpoints {
		t.Run(name, func(t *testing.T) {
			if requiresAdmin {
				t.Fatal("expected metrics endpoint to allow non-admin authenticated users")
			}
		})
	}
}
