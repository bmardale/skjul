package static

import (
	"embed"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//go:embed all:dist
var embedded embed.FS

var spaHandler http.Handler

func init() {
	fsys, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	spaHandler = newSPAHandler(fsys)
}

func Handler() http.Handler {
	return spaHandler
}

func newSPAHandler(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}

		if serveFile(w, r, fsys, p) {
			return
		}

		serveFile(w, r, fsys, "index.html")
	})
}

func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return false
	}

	ct := mime.TypeByExtension(filepath.Ext(name))
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	rs, ok := f.(io.ReadSeeker)
	if ok {
		http.ServeContent(w, r, "", time.Time{}, rs)
	} else {
		w.WriteHeader(http.StatusOK)
		io.Copy(w, f)
	}
	return true
}
