package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeImageStorageEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		defaultSecure bool
		wantEndpoint  string
		wantSecure    bool
		wantError     bool
	}{
		{name: "host defaults to HTTPS", raw: "minio.example.com:9000", defaultSecure: true, wantEndpoint: "minio.example.com:9000", wantSecure: true},
		{name: "explicit HTTP", raw: "http://minio:9000", defaultSecure: true, wantEndpoint: "minio:9000", wantSecure: false},
		{name: "reject path", raw: "https://minio.example.com/storage", wantError: true},
		{name: "reject credentials", raw: "https://user:pass@minio.example.com", wantError: true},
		{name: "reject query", raw: "https://minio.example.com?region=local", wantError: true},
		{name: "reject unsupported scheme", raw: "ftp://minio.example.com", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint, secure, err := normalizeImageStorageEndpoint(test.raw, test.defaultSecure)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantEndpoint, endpoint)
			require.Equal(t, test.wantSecure, secure)
		})
	}
}

func TestValidateImageStorageConfigRejectsParentPrefix(t *testing.T) {
	config := ImageObjectStorageConfig{
		Endpoint:     "https://minio.example.com",
		Bucket:       "julong-media",
		AccessKey:    "access",
		SecretKey:    "secret",
		ObjectPrefix: "generated/../private",
	}
	require.Error(t, validateImageStorageConfig(config, true))
}
