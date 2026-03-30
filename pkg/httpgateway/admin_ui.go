package httpgateway

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var uiFS embed.FS

// AdminUIFS returns the file system containing the admin UI.
func AdminUIFS() http.FileSystem {
	// Extract the "ui" subtree so that the server serves the files directly
	subFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		panic("Failed to initialize embedded UI filesystem: " + err.Error())
	}
	return http.FS(subFS)
}

// ServeAdminUI serves the main admin UI HTML page.
func ServeAdminUI(w http.ResponseWriter, r *http.Request) {
	data, err := uiFS.ReadFile("ui/index.html")
	if err != nil {
		http.Error(w, "Admin UI not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
