package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	imageStorageEnabledKey   = "ImageGenerationStorageEnabled"
	imageStorageEndpointKey  = "ImageGenerationStorageEndpoint"
	imageStorageBucketKey    = "ImageGenerationStorageBucket"
	imageStorageRegionKey    = "ImageGenerationStorageRegion"
	imageStorageAccessKey    = "ImageGenerationStorageAccessKey"
	imageStorageSecretKey    = "ImageGenerationStorageSecret"
	imageStorageUseSSLKey    = "ImageGenerationStorageUseSSL"
	imageStoragePathStyleKey = "ImageGenerationStoragePathStyle"
	imageStoragePrefixKey    = "ImageGenerationStoragePrefix"
)

type ImageObjectStorageConfig struct {
	Enabled      bool   `json:"enabled"`
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key,omitempty"`
	HasSecretKey bool   `json:"has_secret_key"`
	UseSSL       bool   `json:"use_ssl"`
	UsePathStyle bool   `json:"use_path_style"`
	ObjectPrefix string `json:"object_prefix"`
}

type StoredImageObject struct {
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
	SHA256    string `json:"sha256"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
}

func GetImageObjectStorageConfig() ImageObjectStorageConfig {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	config := ImageObjectStorageConfig{
		Enabled:      common.OptionMap[imageStorageEnabledKey] == "true",
		Endpoint:     common.OptionMap[imageStorageEndpointKey],
		Bucket:       common.OptionMap[imageStorageBucketKey],
		Region:       common.OptionMap[imageStorageRegionKey],
		AccessKey:    common.OptionMap[imageStorageAccessKey],
		SecretKey:    common.OptionMap[imageStorageSecretKey],
		UseSSL:       common.OptionMap[imageStorageUseSSLKey] != "false",
		UsePathStyle: common.OptionMap[imageStoragePathStyleKey] != "false",
		ObjectPrefix: common.OptionMap[imageStoragePrefixKey],
	}
	applyImageStorageDefaults(&config)
	config.HasSecretKey = strings.TrimSpace(config.SecretKey) != ""
	return config
}

func PublicImageObjectStorageConfig() ImageObjectStorageConfig {
	config := GetImageObjectStorageConfig()
	config.SecretKey = ""
	return config
}

func SaveImageObjectStorageConfig(config ImageObjectStorageConfig) error {
	existing := GetImageObjectStorageConfig()
	if strings.TrimSpace(config.SecretKey) == "" {
		config.SecretKey = existing.SecretKey
	}
	applyImageStorageDefaults(&config)
	if err := validateImageStorageConfig(config, config.Enabled); err != nil {
		return err
	}
	return model.UpdateOptionsBulk(map[string]string{
		imageStorageEnabledKey:   fmt.Sprintf("%t", config.Enabled),
		imageStorageEndpointKey:  config.Endpoint,
		imageStorageBucketKey:    config.Bucket,
		imageStorageRegionKey:    config.Region,
		imageStorageAccessKey:    config.AccessKey,
		imageStorageSecretKey:    config.SecretKey,
		imageStorageUseSSLKey:    fmt.Sprintf("%t", config.UseSSL),
		imageStoragePathStyleKey: fmt.Sprintf("%t", config.UsePathStyle),
		imageStoragePrefixKey:    config.ObjectPrefix,
	})
}

func TestImageObjectStorage(ctx context.Context, candidate ImageObjectStorageConfig) error {
	existing := GetImageObjectStorageConfig()
	if strings.TrimSpace(candidate.SecretKey) == "" {
		candidate.SecretKey = existing.SecretKey
	}
	applyImageStorageDefaults(&candidate)
	if err := validateImageStorageConfig(candidate, true); err != nil {
		return err
	}
	client, err := imageStorageClient(candidate)
	if err != nil {
		return err
	}
	exists, err := client.BucketExists(ctx, candidate.Bucket)
	if err != nil {
		return fmt.Errorf("连接 MinIO 失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("Bucket %q 不存在", candidate.Bucket)
	}
	return nil
}

func StoreGeneratedImageObject(ctx context.Context, data []byte, mimeType string, extension string) (StoredImageObject, error) {
	config := GetImageObjectStorageConfig()
	if !config.Enabled {
		return StoredImageObject{}, fmt.Errorf("MinIO 未启用")
	}
	if err := validateImageStorageConfig(config, true); err != nil {
		return StoredImageObject{}, err
	}
	client, err := imageStorageClient(config)
	if err != nil {
		return StoredImageObject{}, err
	}
	digest := sha256.Sum256(data)
	sha := hex.EncodeToString(digest[:])
	date := time.Now().UTC().Format("2006/01/02")
	objectKey := path.Join(config.ObjectPrefix, date, sha+extension)
	_, err = client.PutObject(ctx, config.Bucket, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: mimeType,
		UserMetadata: map[string]string{
			"sha256": sha,
			"source": "api.julongkj.top",
		},
	})
	if err != nil {
		return StoredImageObject{}, fmt.Errorf("上传 MinIO 失败: %w", err)
	}
	return StoredImageObject{Bucket: config.Bucket, ObjectKey: objectKey, SHA256: sha, MimeType: mimeType, Size: int64(len(data))}, nil
}

func ReadGeneratedImageObject(ctx context.Context, bucket, objectKey string, maxBytes int64) ([]byte, error) {
	config := GetImageObjectStorageConfig()
	if bucket != "" && bucket != config.Bucket {
		return nil, fmt.Errorf("对象 Bucket 与当前配置不一致")
	}
	client, err := imageStorageClient(config)
	if err != nil {
		return nil, err
	}
	object, err := client.GetObject(ctx, config.Bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()
	data, err := io.ReadAll(io.LimitReader(object, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("对象大小超过限制")
	}
	return data, nil
}

func PresignGeneratedImageObject(ctx context.Context, bucket, objectKey string, expiry time.Duration) (string, time.Time, error) {
	config := GetImageObjectStorageConfig()
	if !config.Enabled {
		return "", time.Time{}, fmt.Errorf("MinIO 未启用")
	}
	if bucket != "" && bucket != config.Bucket {
		return "", time.Time{}, fmt.Errorf("对象 Bucket 与当前配置不一致")
	}
	if strings.TrimSpace(objectKey) == "" {
		return "", time.Time{}, fmt.Errorf("对象路径不能为空")
	}
	if expiry < time.Minute || expiry > 24*time.Hour {
		return "", time.Time{}, fmt.Errorf("临时地址有效期必须在 60 秒到 24 小时之间")
	}
	client, err := imageStorageClient(config)
	if err != nil {
		return "", time.Time{}, err
	}
	presigned, err := client.PresignedGetObject(ctx, config.Bucket, objectKey, expiry, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成临时访问地址失败: %w", err)
	}
	expiresAt := time.Now().Add(expiry)
	return presigned.String(), expiresAt, nil
}

func applyImageStorageDefaults(config *ImageObjectStorageConfig) {
	config.Endpoint = strings.TrimSpace(config.Endpoint)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Region = strings.TrimSpace(config.Region)
	config.AccessKey = strings.TrimSpace(config.AccessKey)
	config.SecretKey = strings.TrimSpace(config.SecretKey)
	config.ObjectPrefix = strings.Trim(strings.TrimSpace(config.ObjectPrefix), "/")
	if config.Bucket == "" {
		config.Bucket = "julong-media"
	}
	if config.Region == "" {
		config.Region = "us-east-1"
	}
	if config.ObjectPrefix == "" {
		config.ObjectPrefix = "generated/images"
	}
}

func validateImageStorageConfig(config ImageObjectStorageConfig, requireCredentials bool) error {
	if strings.TrimSpace(config.Endpoint) == "" {
		if !requireCredentials {
			return nil
		}
		return fmt.Errorf("请填写 MinIO Endpoint")
	}
	if strings.TrimSpace(config.Bucket) == "" {
		return fmt.Errorf("请填写 Bucket")
	}
	if requireCredentials && (strings.TrimSpace(config.AccessKey) == "" || strings.TrimSpace(config.SecretKey) == "") {
		return fmt.Errorf("请填写 Access Key 和 Secret Key")
	}
	for _, segment := range strings.Split(config.ObjectPrefix, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("对象前缀不能包含 . 或 .. 路径段")
		}
	}
	_, _, err := normalizeImageStorageEndpoint(config.Endpoint, config.UseSSL)
	return err
}

func imageStorageClient(config ImageObjectStorageConfig) (*minio.Client, error) {
	endpoint, secure, err := normalizeImageStorageEndpoint(config.Endpoint, config.UseSSL)
	if err != nil {
		return nil, err
	}
	lookup := minio.BucketLookupAuto
	if config.UsePathStyle {
		lookup = minio.BucketLookupPath
	}
	return minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure:       secure,
		Region:       config.Region,
		BucketLookup: lookup,
	})
}

func normalizeImageStorageEndpoint(raw string, defaultSecure bool) (string, bool, error) {
	value := strings.TrimSpace(raw)
	if !strings.Contains(value, "://") {
		value = map[bool]string{true: "https://", false: "http://"}[defaultSecure] + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", false, fmt.Errorf("MinIO Endpoint 格式不正确")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false, fmt.Errorf("MinIO Endpoint 仅支持 HTTP 或 HTTPS")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false, fmt.Errorf("MinIO Endpoint 不能包含凭据、查询参数或片段")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", false, fmt.Errorf("MinIO Endpoint 不能包含路径")
	}
	return parsed.Host, parsed.Scheme == "https", nil
}

func MarshalImageStorageContract(object StoredImageObject) json.RawMessage {
	encoded, _ := json.Marshal(object)
	return encoded
}
