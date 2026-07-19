package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeImageGenerationLogImageAuthWhitelist(t *testing.T) {
	normalized, err := NormalizeImageGenerationLogImageAuthWhitelist(" HTTPS://Example.COM:8443/path, 203.0.113.10\n[2001:db8::1]\nexample.com ")
	require.NoError(t, err)
	require.Equal(t, "example.com\n203.0.113.10\n2001:db8::1", normalized)

	for _, invalid := range []string{"*.example.com", "example.com/path", "bad_domain.example", "ftp://example.com"} {
		_, err = NormalizeImageGenerationLogImageAuthWhitelist(invalid)
		require.Error(t, err, invalid)
	}
}

func TestImageGenerationLogImageAuthWhitelistMatches(t *testing.T) {
	tests := []struct {
		name      string
		whitelist string
		clientIP  string
		origin    string
		referer   string
		want      bool
	}{
		{name: "exact IP", whitelist: "203.0.113.10", clientIP: "203.0.113.10", want: true},
		{name: "different IP", whitelist: "203.0.113.10", clientIP: "203.0.113.11", want: false},
		{name: "origin domain", whitelist: "example.com", origin: "https://EXAMPLE.com:8443", want: true},
		{name: "referer domain", whitelist: "example.com", referer: "https://example.com/gallery/1", want: true},
		{name: "subdomain is not exact", whitelist: "example.com", referer: "https://cdn.example.com/gallery/1", want: false},
		{name: "allow all", whitelist: "0.0.0.0", clientIP: "198.51.100.20", want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, ImageGenerationLogImageAuthWhitelistMatches(
				test.whitelist,
				test.clientIP,
				test.origin,
				test.referer,
			))
		})
	}
}
