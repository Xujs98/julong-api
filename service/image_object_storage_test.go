package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

type fakeImageObjectStorageClient struct {
	objects []minio.ObjectInfo
	removed []string
	err     error
}

func (client *fakeImageObjectStorageClient) ListObjects(_ context.Context, _ string, _ minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	objects := make(chan minio.ObjectInfo, len(client.objects)+1)
	for _, object := range client.objects {
		objects <- object
	}
	if client.err != nil {
		objects <- minio.ObjectInfo{Err: client.err}
	}
	close(objects)
	return objects
}

func (client *fakeImageObjectStorageClient) RemoveObject(_ context.Context, _ string, objectName string, _ minio.RemoveObjectOptions) error {
	client.removed = append(client.removed, objectName)
	return nil
}

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

func TestValidateImageStorageConfigRejectsInvalidRetentionDays(t *testing.T) {
	config := ImageObjectStorageConfig{RetentionDays: -1}
	require.Error(t, validateImageStorageConfig(config, false))
	config.RetentionDays = maxImageStorageRetentionDays + 1
	require.Error(t, validateImageStorageConfig(config, false))
}

func TestParseImageStorageRetentionDays(t *testing.T) {
	require.Equal(t, defaultImageStorageRetentionDays, parseImageStorageRetentionDays(""))
	require.Equal(t, defaultImageStorageRetentionDays, parseImageStorageRetentionDays("invalid"))
	require.Equal(t, 0, parseImageStorageRetentionDays("0"))
	require.Equal(t, 90, parseImageStorageRetentionDays(" 90 "))
}

func TestScanGeneratedImageObjectsDeletesOnlyExpiredObjects(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	client := &fakeImageObjectStorageClient{objects: []minio.ObjectInfo{
		{Key: "generated/images/", LastModified: now.Add(-90 * 24 * time.Hour)},
		{Key: "generated/images/expired.png", Size: 100, LastModified: now.Add(-31 * 24 * time.Hour)},
		{Key: "generated/images/current.png", Size: 200, LastModified: now.Add(-29 * 24 * time.Hour)},
		{Key: "generated/images/unknown.png", Size: 300},
	}}
	config := ImageObjectStorageConfig{
		Bucket:        "julong-media",
		ObjectPrefix:  "generated/images",
		RetentionDays: 30,
	}

	result, err := scanGeneratedImageObjects(context.Background(), client, config, now, true)

	require.NoError(t, err)
	require.Equal(t, int64(3), result.FileCount)
	require.Equal(t, int64(600), result.TotalSize)
	require.Equal(t, int64(1), result.ExpiredCount)
	require.Equal(t, int64(100), result.ExpiredSize)
	require.Equal(t, int64(1), result.DeletedCount)
	require.Equal(t, int64(100), result.DeletedBytes)
	require.Equal(t, []string{"generated/images/expired.png"}, client.removed)
}

func TestScanGeneratedImageObjectsReturnsListingError(t *testing.T) {
	client := &fakeImageObjectStorageClient{err: errors.New("list failed")}
	config := ImageObjectStorageConfig{
		Bucket:        "julong-media",
		ObjectPrefix:  "generated/images",
		RetentionDays: 30,
	}

	_, err := scanGeneratedImageObjects(context.Background(), client, config, time.Now(), false)

	require.ErrorContains(t, err, "list failed")
}
