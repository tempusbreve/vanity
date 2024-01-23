package handlers

import (
	"html/template"
	"net/http"
	"net/url"
)

// Import describes an import reference in Go.
type Import struct {
	// Prefix, import-prefix, is the import path corresponding to the repository root.
	Prefix string `json:"prefix"`
	// VCS is one of bzr, fossil, git, hg, svn.
	VCS string `json:"vcs"`
	// Root is the version control system root; https://example.org/foo/bar/proj
	Root string `json:"root"`
	// Proxy indicates an optional proxy variant
	Proxy string `json:"proxy"`
}

// ImportStore describes the interface to lookup Import records.
type ImportStore interface {
	// Lookup uses a URL to find the Import if any.
	Lookup(*url.URL) (Import, bool)
}

// ImportHandler is the http.Handler for responding to import requests.
type ImportHandler struct {
	fallback http.Handler
	store    ImportStore
	tmpl     *template.Template
}

func (i *ImportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lookup := r.URL
	lookup.Host = r.Host

	if r.URL.Path != "" || true {
		if imp, ok := i.store.Lookup(lookup); ok {
			fromGo := r.URL.Query().Get("go-get") == "1"

			w.WriteHeader(http.StatusOK)

			if err := i.tmpl.Execute(w, &importData{Import: imp, FromGO: fromGo}); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			return
		}
	}

	if i.fallback != nil {
		i.fallback.ServeHTTP(w, r)

		return
	}

	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

var _ http.Handler = (*ImportHandler)(nil)

// NewImportHandler generates an ImportHandler with fallback and optional stores.
func NewImportHandler(fallback http.Handler, stores ...ImportStore) *ImportHandler {
	return &ImportHandler{
		fallback: fallback,
		store:    importStores(stores),
		tmpl: template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
  <head>{{ if .Import.Proxy }}
    <meta name="go-import" content="{{ .Import.Prefix }} mod {{ .Import.Proxy }}">{{ else }}
    <meta name="go-import" content="{{ .Import.Prefix }} {{ .Import.VCS }} {{ .Import.Root }}">{{ end }}
    {{- if .FromGO }}{{ else }}
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta http-equiv="refresh" content="10; url=https://godoc.org/{{ .Import.Prefix }}" />{{ end }}
  </head>
  <body>{{ if .FromGO }}{{ else }}
    <div>
      <h1>{{ .Import.Prefix }} Found</h1>
      <p>Documentation at <a href="https://godoc.org/{{ .Import.Prefix }}">godoc.org/{{ .Import.Prefix }}</a></p>
      <p>Redirecting . . .</p>
    </div>{{ end }}
  </body>
</html>`)),
	}
}

type importData struct {
	Import Import
	FromGO bool
}

type importStores []ImportStore

func (s importStores) Lookup(u *url.URL) (Import, bool) {
	for _, instance := range s {
		if imp, ok := instance.Lookup(u); ok {
			return imp, ok
		}
	}

	return Import{}, false
}
