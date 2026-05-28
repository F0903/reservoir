package services

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"reservoir/db"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	coreauth "reservoir/webserver/auth"
)

func newAuthTestStore(t *testing.T) *stores.UserStore {
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

func createAuthTestUser(t *testing.T, store *stores.UserStore) *models.User {
	t.Helper()

	user, err := store.Create(&models.User{
		Username:     "operator",
		PasswordHash: *phc.GenerateArgon2id("current-password"),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func TestLoginCookieSecureFollowsRequestContext(t *testing.T) {
	tests := []struct {
		name        string
		requestURL  string
		headerName  string
		headerValue string
		wantSecure  bool
	}{
		{
			name:       "plain HTTP request",
			requestURL: "http://reservoir.test/api/auth/login",
		},
		{
			name:       "direct HTTPS request",
			requestURL: "https://reservoir.test/api/auth/login",
			wantSecure: true,
		},
		{
			name:        "forwarded proto HTTPS request",
			requestURL:  "http://reservoir.test/api/auth/login",
			headerName:  "X-Forwarded-Proto",
			headerValue: "https",
			wantSecure:  true,
		},
		{
			name:        "RFC forwarded HTTPS request",
			requestURL:  "http://reservoir.test/api/auth/login",
			headerName:  "Forwarded",
			headerValue: `for=192.0.2.1;proto=https;host=reservoir.test`,
			wantSecure:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newAuthTestStore(t)
			createAuthTestUser(t, store)
			svc := &AuthService{users: store, sessions: coreauth.NewSessionManager()}

			req := httptest.NewRequest(http.MethodPost, tt.requestURL, nil)
			if tt.headerName != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}
			reqCtx, authenticated := svc.RequestFromHTTP(req)
			if authenticated {
				t.Fatal("expected login request to be unauthenticated")
			}

			_, cookie, err := svc.Login(reqCtx, "operator", "current-password")
			if err != nil {
				t.Fatalf("Login returned error: %v", err)
			}
			if cookie.Secure != tt.wantSecure {
				t.Fatalf("expected cookie Secure=%v, got %v", tt.wantSecure, cookie.Secure)
			}
		})
	}
}

func TestCreateBootstrapAdminSetsSecureCookieForHTTPSRequest(t *testing.T) {
	store := newAuthTestStore(t)
	svc := &AuthService{users: store, sessions: coreauth.NewSessionManager()}
	req := httptest.NewRequest(http.MethodPost, "https://reservoir.test/api/auth/bootstrap", nil)

	reqCtx, authenticated := svc.RequestFromHTTP(req)
	if authenticated {
		t.Fatal("expected bootstrap request to be unauthenticated")
	}

	_, cookie, err := svc.CreateBootstrapAdmin(reqCtx, "admin", "generated-password")
	if err != nil {
		t.Fatalf("CreateBootstrapAdmin returned error: %v", err)
	}
	if !cookie.Secure {
		t.Fatal("expected bootstrap session cookie to be Secure")
	}
}

func TestLogoutClearsSecureCookieForHTTPSRequest(t *testing.T) {
	sessions := coreauth.NewSessionManager()
	session := sessions.Create(1)
	svc := &AuthService{sessions: sessions}
	req := httptest.NewRequest(http.MethodPost, "https://reservoir.test/api/auth/logout", nil)
	req.AddCookie(session.BuildSessionCookieWithSecure(true))

	reqCtx, authenticated := svc.RequestFromHTTP(req)
	if !authenticated {
		t.Fatal("expected logout request to be authenticated")
	}

	cookie, err := svc.Logout(reqCtx)
	if err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if !cookie.Secure {
		t.Fatal("expected logout deletion cookie to be Secure")
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("expected logout cookie MaxAge -1, got %d", cookie.MaxAge)
	}
}

func TestChangePasswordRejectsShortNewPassword(t *testing.T) {
	store := newAuthTestStore(t)
	user, err := store.Create(&models.User{
		Username:               "operator",
		PasswordHash:           *phc.GenerateArgon2id("current-password"),
		IsAdmin:                true,
		PasswordChangeRequired: false,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessions := coreauth.NewSessionManager()
	session := sessions.Create(user.ID)
	svc := &AuthService{users: store, sessions: sessions}

	err = svc.ChangePassword(RequestContext{session: session}, "current-password", "short")
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}

	unchanged, err := store.GetByID(user.ID)
	if err != nil {
		t.Fatalf("failed to reload test user: %v", err)
	}
	if !unchanged.PasswordHash.VerifyArgon2id("current-password") {
		t.Fatal("short password update changed the stored password")
	}
}
