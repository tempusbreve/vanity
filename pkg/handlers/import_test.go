package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tempusbreve/vanity/pkg/handlers"
)

var jsonData = `[
{"prefix":"example.org/tempusbreve/vanity","vcs":"git","root":"https://github.com/tempusbreve/vanity","proxy":""},
{"prefix":"example.org/tempusbreve/proxy","vcs":"git","root":"https://github.com/tempusbreve/proxy","proxy":"https://proxy.golang.org/"},
{}]`

func rawJSON() (io.Reader, func(), error) { return strings.NewReader(jsonData), func() {}, nil }

func TestImportHandler_ServeHTTP(t *testing.T) {
	for storeName, store := range map[string]handlers.ImportStore{
		"testStore": &testStore{
			"https://example.org/tempusbreve/vanity": {
				Prefix: "example.org/tempusbreve/vanity",
				VCS:    "git",
				Root:   "https://github.com/tempusbreve/vanity",
				Proxy:  "",
			},
			"https://example.org/tempusbreve/proxy": {
				Prefix: "example.org/tempusbreve/proxy",
				VCS:    "git",
				Root:   "https://github.com/tempusbreve/proxy",
				Proxy:  "https://proxy.golang.org/",
			},
		},
		"testDNSStore": handlers.NewDNSStore(testResolver{
			"example.org": []string{
				"go-import=example.org/tempusbreve/vanity git https://github.com/tempusbreve/vanity",
				"go-import=example.org/tempusbreve/proxy git https://github.com/tempusbreve/proxy https://proxy.golang.org/",
			},
		}),
		"testJSONStore": handlers.NewJSONStore(rawJSON),
	} {
		for name, tc := range map[string]testCase{
			"empty":      {method: http.MethodGet, url: "", expect: http.StatusNotFound},
			"bare":       {method: http.MethodGet, url: "https://example.org/", expect: http.StatusNotFound},
			"protected":  {method: http.MethodGet, url: "https://example.com/foo/bar", expect: http.StatusUnauthorized},
			"missing":    {method: http.MethodGet, url: "https://example.org/foo/bar", expect: http.StatusNotFound},
			"vanity":     {method: http.MethodGet, url: "https://example.org/tempusbreve/vanity?go-get=1", expect: http.StatusOK},
			"proxy":      {method: http.MethodGet, url: "https://example.org/tempusbreve/proxy?go-get=1", expect: http.StatusOK},
			"google":     {method: http.MethodGet, url: "https://google.com/package/name", expect: http.StatusNotFound},
			"cloudflare": {method: http.MethodGet, url: "https://cloudflare.com/package/name", expect: http.StatusNotFound},
			"golang":     {method: http.MethodGet, url: "https://golang.org/package/name", expect: http.StatusNotFound},
		} {
			cs := tc

			t.Run(storeName+" // "+name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, cs.method, cs.url, nil)
				if err != nil {
					t.Error(err)
				}

				rr := httptest.NewRecorder()
				handler := handlers.NewImportHandler(&testFallback{}, store)

				handler.ServeHTTP(rr, req)

				if status := rr.Code; status != cs.expect {
					t.Errorf("wrong status; expected %d, got %d", cs.expect, status)
				}
			})
		}
	}
}

type testStore map[string]handlers.Import

func (s testStore) Lookup(u *url.URL) (handlers.Import, bool) {
	key := u.Scheme + "://" + u.Host + u.Path
	if imp, ok := s[key]; ok {
		return imp, true
	}

	return handlers.Import{}, false
}

type testFallback struct{}

func (testFallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Host == "example.com" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

		return
	}

	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

type testCase struct {
	method string
	url    string
	expect int
}
