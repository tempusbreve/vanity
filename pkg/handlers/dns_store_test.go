package handlers_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/tempusbreve/vanity/pkg/handlers"
)

func TestDNSStore(t *testing.T) {
	store := handlers.NewDNSStore(testResolver{
		"example.org": []string{
			"go-import=example.org/one git https://example.com/org/one",
			"go-import=example.org/two git https://example.com/org/two",
			"foo=bar",
		},
		"example.com": []string{
			"baz=quux",
		},
	})

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

type testResolver map[string][]string

func (r testResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	if v, ok := r[name]; ok {
		return v, nil
	}

	return nil, fmt.Errorf("not implemented")
}
