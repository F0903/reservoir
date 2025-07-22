package endpoints

import (
	"net/http"
)

type SettingsEndpoint struct{}

func (m *SettingsEndpoint) Path() string {
	return "/settings"
}

func (m *SettingsEndpoint) Method() string {
	return "GET"
}

func (m *SettingsEndpoint) Endpoint(w http.ResponseWriter, r *http.Request) {

}
