package handlers_test

import (
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/tempusbreve/vanity/pkg/handlers"
)

func TestJSONStore(t *testing.T) {
	data := `[
{"prefix":"example.org/one","vcs":"git","root":"https://example.com/org/one","proxy":""},
{"prefix":"example.org/two","vcs":"git","root":"https://example.com/org/two","proxy":""},
{}]`
	dataFn := func() (io.Reader, func(), error) { return strings.NewReader(data), func() {}, nil }

	store := handlers.NewJSONStore(dataFn)
	for l, shouldPass := range map[string]bool{
		"http://example.org/one?go-get=1":     true,
		"http://example.org/two":              true,
		"http://example.org/authsvc?go-get=1": false,
		"http://example.com/one":              false,
	} {
		str := l
		t.Run(l, func(t *testing.T) {
			u, err := url.Parse(str)
			if err != nil {
				t.Fatal(err)
			}

			i, ok := store.Lookup(u)
			if shouldPass {
				if !ok {
					t.Errorf("not found: %q", l)
					return
				}

				if i.Prefix == "" {
					t.Errorf("no prefix: %q", l)
				}
			} else {
				if ok {
					t.Errorf("found unexpectedly: %q", l)
				}
			}
		})
	}
}
