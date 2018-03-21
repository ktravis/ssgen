package main

import (
	"net/http"
	"os"
	"path"
	"strings"
)

var (
	indexPages = []string{
		"index.html",
		"index.htm",
		"index.txt",
		"default.html",
		"default.htm",
		"default.txt",
	}
)

type FileServer struct {
	http.FileSystem
}

func writeError(w http.ResponseWriter, err error) {
	if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	} else if os.IsPermission(err) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("permission denied"))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func redir(w http.ResponseWriter, r *http.Request, p string) {
	for strings.HasPrefix(p, "//") {
		p = strings.TrimPrefix(p, "/")
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func (fs FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	f, err := fs.Open(p)
	if err != nil {
		writeError(w, err)
		return
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		writeError(w, err)
		return
	}

	if d.IsDir() {
		if !strings.HasSuffix(p, "/") {
			redir(w, r, p+"/")
			return
		}
		for _, idx := range indexPages {
			f, err := fs.Open(path.Join(p, idx))
			if err != nil {
				continue
			}
			defer f.Close()

			d, err := f.Stat()
			if err != nil {
				f.Close()
				continue
			}

			if d.IsDir() {
				writeError(w, os.ErrNotExist)
			} else {
				http.ServeContent(w, r, d.Name(), d.ModTime(), f)
			}
			return
		}
		writeError(w, os.ErrNotExist)
		return
	}

	if strings.HasSuffix(p, "/") {
		redir(w, r, strings.TrimSuffix(p, "/"))
		return
	}
	b := path.Base(p)
	for _, idx := range indexPages {
		if idx == b {
			redir(w, r, strings.TrimSuffix(p, idx))
			return
		}
	}
	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}
