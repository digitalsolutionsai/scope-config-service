package httpgateway

import (
	"net/http"
	_ "embed"
)

//go:embed admin.html
var adminHTML []byte

// ServeAdminUI serves the built-in admin UI HTML page.
func ServeAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(adminHTML)
}
