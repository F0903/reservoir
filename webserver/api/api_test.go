package api

import (
	"testing"

	dbmodels "reservoir/db/models"
	"reservoir/webserver/api/apitypes"
	cacheEndpoint "reservoir/webserver/api/endpoints/cache"
	"reservoir/webserver/api/endpoints/log"
	"reservoir/webserver/api/endpoints/metrics"
)

func TestPasswordChangeAllowed(t *testing.T) {
	tests := []struct {
		name   string
		user   *dbmodels.User
		method apitypes.EndpointMethod
		want   bool
	}{
		{
			name:   "BlocksRegularEndpoints",
			user:   &dbmodels.User{PasswordChangeRequired: true},
			method: apitypes.EndpointMethod{RequiresAuth: true},
			want:   false,
		},
		{
			name:   "AllowsExplicitAuthEndpoints",
			user:   &dbmodels.User{PasswordChangeRequired: true},
			method: apitypes.EndpointMethod{RequiresAuth: true, AllowPasswordChangeRequired: true},
			want:   true,
		},
		{
			name:   "AllowsConfiguredUsers",
			user:   &dbmodels.User{PasswordChangeRequired: false},
			method: apitypes.EndpointMethod{RequiresAuth: true},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := passwordChangeAllowed(tt.user, tt.method); got != tt.want {
				t.Fatalf("passwordChangeAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdminAllowed(t *testing.T) {
	tests := []struct {
		name   string
		user   *dbmodels.User
		method apitypes.EndpointMethod
		want   bool
	}{
		{
			name:   "BlocksNonAdminsWhenConfigured",
			user:   &dbmodels.User{IsAdmin: false},
			method: apitypes.EndpointMethod{RequiresAuth: true, RequiresAdmin: true},
			want:   false,
		},
		{
			name:   "AllowsAdmins",
			user:   &dbmodels.User{IsAdmin: true},
			method: apitypes.EndpointMethod{RequiresAuth: true, RequiresAdmin: true},
			want:   true,
		},
		{
			name:   "AllowsRegularEndpoints",
			user:   &dbmodels.User{IsAdmin: false},
			method: apitypes.EndpointMethod{RequiresAuth: true},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adminAllowed(tt.user, tt.method); got != tt.want {
				t.Fatalf("adminAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpointAdminRequirements(t *testing.T) {
	tests := []struct {
		name              string
		method            apitypes.EndpointMethod
		wantRequiresAdmin bool
	}{
		{name: "cache status", method: (&cacheEndpoint.StatusEndpoint{}).EndpointMethods()[0]},
		{name: "cache clear", method: (&cacheEndpoint.ClearEndpoint{}).EndpointMethods()[0], wantRequiresAdmin: true},
		{name: "log", method: (&log.LogEndpoint{}).EndpointMethods()[0]},
		{name: "log stream", method: (&log.LogStreamEndpoint{}).EndpointMethods()[0]},
		{name: "metrics all", method: (&metrics.AllMetricsEndpoint{}).EndpointMethods()[0]},
		{name: "metrics cache", method: (&metrics.CacheMetricsEndpoint{}).EndpointMethods()[0]},
		{name: "metrics requests", method: (&metrics.RequestsMetricsEndpoint{}).EndpointMethods()[0]},
		{name: "metrics system", method: (&metrics.SystemMetricsEndpoint{}).EndpointMethods()[0]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.method.RequiresAuth {
				t.Fatal("expected endpoint to require authentication")
			}
			if tt.method.RequiresAdmin != tt.wantRequiresAdmin {
				t.Fatalf("expected RequiresAdmin=%t, got %t", tt.wantRequiresAdmin, tt.method.RequiresAdmin)
			}
		})
	}
}
