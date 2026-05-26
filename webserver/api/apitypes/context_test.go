package apitypes

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	appservices "reservoir/application/services"
	"reservoir/config"
	"reservoir/db"
	"reservoir/db/stores"
	"reservoir/webserver/auth"
)

func newTestContextUserStore(t *testing.T) *stores.UserStore {
	t.Helper()

	databasePath := filepath.ToSlash(filepath.Join(t.TempDir(), "database.db"))
	database, err := db.Open(databasePath, 5000)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	store := stores.NewUserStore(database)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close test user store: %v", err)
		}
	})
	return store
}

func TestCreateContextCarriesInjectedUserStoreForUnauthenticatedRequests(t *testing.T) {
	users := newTestContextUserStore(t)
	services := appservices.NewContainer(config.NewDefault(), auth.NewSessionManager(), users, nil)
	req := httptest.NewRequest("POST", "/api/auth/login", nil)

	ctx, err := CreateContext(req, services)
	if err != nil {
		t.Fatalf("CreateContext returned error: %v", err)
	}

	if ctx.IsAuthenticated() {
		t.Fatal("expected request without session cookie to be unauthenticated")
	}
	if ctx.Services != services {
		t.Fatal("expected context to use injected application services")
	}
}
