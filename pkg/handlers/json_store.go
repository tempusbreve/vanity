package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
)

// JSONStore implements an ImportStore reading from JSON formatted file.
type JSONStore struct {
	src func() (io.Reader, func(), error)
}

func NewFileReader(path string) func() (io.Reader, func(), error) {
	return func() (io.Reader, func(), error) {
		if fd, err := os.Open(path); err != nil {
			return nil, nil, err
		} else {
			return fd, func() { fd.Close() }, nil
		}
	}
}

// NewJSONStore creates an initialized JSONStore.
func NewJSONStore(src func() (io.Reader, func(), error)) *JSONStore { return &JSONStore{src: src} }

// Lookup fulfills the interface.
func (s *JSONStore) Lookup(u *url.URL) (Import, bool) {
	in, cleanup, err := s.src()
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSONStore.Lookup(%q): open error: %v\n", u, err)
		return Import{}, false
	}
	defer cleanup()

	var recs []Import
	if err = json.NewDecoder(in).Decode(&recs); err != nil {
		fmt.Fprintf(os.Stderr, "JSONStore.Lookup(%q): decode error: %v\n", u, err)
		return Import{}, false
	}

	for _, rec := range recs {
		if matchRecord(u, rec) {
			return rec, true
		}
	}

	return Import{}, false
}

func matchRecord(u *url.URL, rec Import) bool {
	return rec.Prefix == u.Host+u.Path
}
