package api

import (
	"testing"

	dbmodels "reservoir/db/models"
	"reservoir/webserver/api/apitypes"
)

func TestPasswordChangeAllowedBlocksRegularEndpoints(t *testing.T) {
	user := &dbmodels.User{PasswordChangeRequired: true}
	method := apitypes.EndpointMethod{RequiresAuth: true}

	if passwordChangeAllowed(user, method) {
		t.Fatal("expected regular endpoint to be blocked while password change is required")
	}
}

func TestPasswordChangeAllowedAllowsExplicitAuthEndpoints(t *testing.T) {
	user := &dbmodels.User{PasswordChangeRequired: true}
	method := apitypes.EndpointMethod{RequiresAuth: true, AllowPasswordChangeRequired: true}

	if !passwordChangeAllowed(user, method) {
		t.Fatal("expected explicitly allowed endpoint to pass while password change is required")
	}
}

func TestPasswordChangeAllowedAllowsConfiguredUsers(t *testing.T) {
	user := &dbmodels.User{PasswordChangeRequired: false}
	method := apitypes.EndpointMethod{RequiresAuth: true}

	if !passwordChangeAllowed(user, method) {
		t.Fatal("expected configured user to access regular endpoint")
	}
}

func TestAdminAllowedBlocksNonAdminsWhenConfigured(t *testing.T) {
	user := &dbmodels.User{IsAdmin: false}
	method := apitypes.EndpointMethod{RequiresAuth: true, RequiresAdmin: true}

	if adminAllowed(user, method) {
		t.Fatal("expected non-admin to be blocked from admin endpoint")
	}
}

func TestAdminAllowedAllowsAdmins(t *testing.T) {
	user := &dbmodels.User{IsAdmin: true}
	method := apitypes.EndpointMethod{RequiresAuth: true, RequiresAdmin: true}

	if !adminAllowed(user, method) {
		t.Fatal("expected admin to be allowed")
	}
}

func TestAdminAllowedAllowsRegularEndpoints(t *testing.T) {
	user := &dbmodels.User{IsAdmin: false}
	method := apitypes.EndpointMethod{RequiresAuth: true}

	if !adminAllowed(user, method) {
		t.Fatal("expected regular endpoint to be allowed")
	}
}
