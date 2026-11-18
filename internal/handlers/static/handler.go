package static

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func New(directory string) http.HandlerFunc {
	files := http.FileServer(http.Dir(directory))
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Path
		if name == "/" {
			name = "/index.html"
		}
		switch strings.ToLower(filepath.Ext(name)) {
		case ".html", ".css", ".js", ".svg", ".jpg", ".jpeg", ".png", ".webp", ".woff", ".woff2", ".ttf", ".txt", ".ico":
		default:
			http.NotFound(w, r)
			return
		}
		info, err := os.Stat(filepath.Join(directory, filepath.FromSlash(strings.TrimPrefix(name, "/"))))
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}
}
