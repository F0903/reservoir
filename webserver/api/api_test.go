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
