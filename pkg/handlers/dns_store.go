package handlers

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	minRecordLength = 3
	timeout         = 15 * time.Second
)

type Resolver interface {
	LookupTXT(context.Context, string) ([]string, error)
}

// DNSImportStore looks up import records from DNS.
type DNSImportStore struct {
	resolver Resolver
}

// NewDNSStore returns an initialized DNSImportStore.
func NewDNSStore(resolver Resolver) *DNSImportStore {
	if resolver == nil {
		resolver = &net.Resolver{PreferGo: true}
	}

	return &DNSImportStore{resolver: resolver}
}

// Lookup fulfills the interface.
func (s *DNSImportStore) Lookup(u *url.URL) (Import, bool) {
	deadline, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	records, err := s.resolver.LookupTXT(deadline, u.Host)
	if err != nil {
		return Import{}, false
	}

	for _, record := range records {
		pair := strings.SplitN(record, "=", 2)
		if pair[0] != "go-import" {
			continue
		}

		if !strings.HasPrefix(pair[1], u.Host+u.Path) {
			continue
		}

		rec := strings.SplitN(pair[1], " ", 4)
		if len(rec) < minRecordLength {
			continue
		}

		if !strings.EqualFold(rec[0], u.Host+u.Path) {
			continue
		}

		imp := Import{Prefix: rec[0], VCS: rec[1], Root: rec[2]}

		if len(rec) > minRecordLength {
			imp.Proxy = rec[3]
		}

		return imp, true
	}

	return Import{}, false
}
