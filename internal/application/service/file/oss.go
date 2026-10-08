package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/Tencent/WeKnora/internal/utils"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/google/uuid"
)

// ossFileService implements the FileService interface for Aliyun OSS
// using the official Aliyun OSS SDK v2 (github.com/aliyun/alibabacloud-oss-go-sdk-v2).
type ossFileService struct {
	client         *oss.Client
	tempClient     *oss.Client
	publicClient   *oss.Client
	pathPrefix     string
	bucketName     string
	tempBucketName string
}

const (
	ossScheme                         = "oss://"
	ossBrowserPreviewCORSMaxRules     = 10
	ossBrowserPreviewCORSCacheSeconds = int64(3600)
)

type ossBrowserPreviewCORSState struct {
	once sync.Once
	err  error
}

var ossBrowserPreviewCORSChecks sync.Map

// newOSSClient creates an OSS client using the official Aliyun SDK v2.
func newOSSClient(endpoint, region, accessKey, secretKey string) (*oss.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(creds).
		WithRegion(region).
		WithEndpoint(endpoint)

	return oss.NewClient(cfg), nil
}

func newOSSCNAMEClient(endpoint, region, accessKey, secretKey string) (*oss.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(creds).
		WithRegion(region).
		WithEndpoint(endpoint).
		WithUseCName(true)

	return oss.NewClient(cfg), nil
}

func ossPublicEndpointForBucket(bucketName string) string {
	endpoint := strings.TrimSpace(os.Getenv("OSS_PUBLIC_ENDPOINT"))
	if endpoint == "" {
		return ""
	}

	publicBucketName := strings.TrimSpace(os.Getenv("OSS_PUBLIC_BUCKET_NAME"))
	if publicBucketName == "" {
		publicBucketName = strings.TrimSpace(os.Getenv("OSS_BUCKET_NAME"))
	}
	if publicBucketName != "" && publicBucketName != bucketName {
		return ""
	}
	return endpoint
}

// ossEnsureBucket checks if the bucket exists and creates it if missing.
func ossEnsureBucket(client *oss.Client, bucketName string) error {
	exists, err := ossBucketExists(context.Background(), client, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check OSS bucket: %w", err)
	}
	if exists {
		return nil
	}

	_, err = client.PutBucket(context.Background(), &oss.PutBucketRequest{
		Bucket: oss.Ptr(bucketName),
	})
	if err != nil {
		var svcErr *oss.ServiceError
		if errors.As(err, &svcErr) && svcErr.StatusCode == http.StatusConflict {
			return nil
		}
		return fmt.Errorf("failed to create OSS bucket: %w", err)
	}
	return nil
}

// ossBucketExists preserves the SDK's permission-tolerant existence check while
// rejecting authentication failures. IsBucketExist treats every OSS service
// error except NoSuchBucket as an existing bucket, including bad signatures.
func ossBucketExists(ctx context.Context, client *oss.Client, bucketName string) (bool, error) {
	_, err := client.GetBucketAcl(ctx, &oss.GetBucketAclRequest{Bucket: oss.Ptr(bucketName)})
	if err == nil {
		return true, nil
	}

	var svcErr *oss.ServiceError
	if !errors.As(err, &svcErr) {
		return false, err
	}

	if ossIsAuthenticationError(err) {
		return false, fmt.Errorf("OSS authentication failed (%s): %w", svcErr.Code, err)
	}

	switch svcErr.Code {
	case "NoSuchBucket":
		return false, nil
	default:
		// A valid key without GetBucketAcl permission returns AccessDenied. The
		// bucket still exists and object-level permissions can be more specific.
		return true, nil
	}
}

func ossIsAuthenticationError(err error) bool {
	var svcErr *oss.ServiceError
	if !errors.As(err, &svcErr) {
		return false
	}
	switch svcErr.Code {
	case "SignatureDoesNotMatch", "InvalidAccessKeyId", "InvalidSecurityToken", "SecurityTokenExpired":
		return true
	default:
		return false
	}
}

func ossBrowserPreviewCORSEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OSS_BROWSER_PREVIEW_CORS_ENABLED"))) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func ossBrowserPreviewAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("OSS_BROWSER_PREVIEW_ALLOWED_ORIGINS"))
	if raw == "" {
		return []string{"*"}
	}

	seen := make(map[string]struct{})
	origins := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(value)
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func ossCORSRuleAllowsBrowserRead(rule oss.CORSRule, origin string) bool {
	allowsOrigin := false
	for _, allowedOrigin := range rule.AllowedOrigins {
		if allowedOrigin == "*" || strings.EqualFold(strings.TrimSpace(allowedOrigin), origin) {
			allowsOrigin = true
			break
		}
	}
	if !allowsOrigin {
		return false
	}
	for _, method := range rule.AllowedMethods {
		if strings.EqualFold(strings.TrimSpace(method), http.MethodGet) {
			return true
		}
	}
	return false
}

func ossCORSAllowsBrowserPreview(config *oss.CORSConfiguration, origins []string) bool {
	if config == nil {
		return false
	}
	for _, origin := range origins {
		allowed := false
		for _, rule := range config.CORSRules {
			if ossCORSRuleAllowsBrowserRead(rule, origin) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func appendOSSBrowserPreviewCORSRule(
	config *oss.CORSConfiguration,
	origins []string,
) (*oss.CORSConfiguration, bool, error) {
	if ossCORSAllowsBrowserPreview(config, origins) {
		return config, false, nil
	}

	next := &oss.CORSConfiguration{}
	if config != nil {
		next.CORSRules = append(next.CORSRules, config.CORSRules...)
		next.ResponseVary = config.ResponseVary
	}
	if len(next.CORSRules) >= ossBrowserPreviewCORSMaxRules {
		return nil, false, fmt.Errorf(
			"OSS bucket already has %d CORS rules; remove or merge a rule before enabling direct preview",
			len(next.CORSRules),
		)
	}

	next.CORSRules = append(next.CORSRules, oss.CORSRule{
		AllowedOrigins: origins,
		AllowedMethods: []string{http.MethodGet, http.MethodHead},
		AllowedHeaders: []string{"*"},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Range",
			"Accept-Ranges",
			"Content-Type",
			"ETag",
			"Last-Modified",
		},
		MaxAgeSeconds: oss.Ptr(ossBrowserPreviewCORSCacheSeconds),
	})
	return next, true, nil
}

func ensureOSSBrowserPreviewCORS(
	ctx context.Context,
	client *oss.Client,
	bucketName string,
	origins []string,
) error {
	result, err := client.GetBucketCors(ctx, &oss.GetBucketCorsRequest{
		Bucket: oss.Ptr(bucketName),
	})

	var config *oss.CORSConfiguration
	if err != nil {
		var svcErr *oss.ServiceError
		if !errors.As(err, &svcErr) || svcErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("get OSS bucket CORS: %w", err)
		}
	} else if result != nil {
		config = result.CORSConfiguration
	}

	next, changed, err := appendOSSBrowserPreviewCORSRule(config, origins)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	_, err = client.PutBucketCors(ctx, &oss.PutBucketCorsRequest{
		Bucket:            oss.Ptr(bucketName),
		CORSConfiguration: next,
	})
	if err != nil {
		return fmt.Errorf("put OSS bucket CORS: %w", err)
	}
	return nil
}

func ensureOSSBrowserPreviewCORSOnce(
	ctx context.Context,
	client *oss.Client,
	bucketName string,
) error {
	origins := ossBrowserPreviewAllowedOrigins()
	cacheKey := bucketName + "|" + strings.Join(origins, ",")
	value, _ := ossBrowserPreviewCORSChecks.LoadOrStore(cacheKey, &ossBrowserPreviewCORSState{})
	state := value.(*ossBrowserPreviewCORSState)
	state.once.Do(func() {
		checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		state.err = ensureOSSBrowserPreviewCORS(checkCtx, client, bucketName, origins)
	})
	return state.err
}

// NewOssFileService creates an Aliyun OSS file service.
// It verifies that the bucket exists and creates it if missing.
func NewOssFileService(endpoint, region, accessKey, secretKey, bucketName, pathPrefix string) (interfaces.FileService, error) {
	return NewOssFileServiceWithTempBucket(endpoint, region, accessKey, secretKey, bucketName, pathPrefix, "", "")
}

// NewOssFileServiceWithTempBucket creates an Aliyun OSS file service with optional temp bucket.
func NewOssFileServiceWithTempBucket(endpoint, region, accessKey, secretKey, bucketName, pathPrefix, tempBucketName, tempRegion string) (interfaces.FileService, error) {
	client, err := newOSSClient(endpoint, region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	if err := ossEnsureBucket(client, bucketName); err != nil {
		return nil, err
	}

	var tempClient *oss.Client
	if tempBucketName != "" {
		if tempRegion == "" {
			tempRegion = region
		}
		tempClient, err = newOSSClient(endpoint, tempRegion, accessKey, secretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OSS temp client: %w", err)
		}
		if err := ossEnsureBucket(tempClient, tempBucketName); err != nil {
			return nil, err
		}
	}

	var publicClient *oss.Client
	if publicEndpoint := ossPublicEndpointForBucket(bucketName); publicEndpoint != "" {
		publicClient, err = newOSSCNAMEClient(publicEndpoint, region, accessKey, secretKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OSS public CNAME client: %w", err)
		}
	}

	// Normalize pathPrefix: ensure it ends with '/' if not empty
	if pathPrefix != "" && !strings.HasSuffix(pathPrefix, "/") {
		pathPrefix += "/"
	}

	return &ossFileService{
		client:         client,
		tempClient:     tempClient,
		publicClient:   publicClient,
		pathPrefix:     pathPrefix,
		bucketName:     bucketName,
		tempBucketName: tempBucketName,
	}, nil
}

// OssFileMigrator uploads existing local files to one concrete OSS backend.
// It is intentionally separate from FileService because migration must keep a
// deterministic object key and must not create a second application resource.
type OssFileMigrator struct {
	service *ossFileService
}

// NewOssFileMigrator creates an OSS uploader for storage migration.
func NewOssFileMigrator(config types.StorageBackendConfig) (*OssFileMigrator, error) {
	client, err := newOSSClient(
		strings.TrimSpace(config.Endpoint),
		strings.TrimSpace(config.Region),
		strings.TrimSpace(config.AccessKeyID),
		strings.TrimSpace(config.SecretAccessKey),
	)
	if err != nil {
		return nil, err
	}
	if err := ossEnsureBucket(client, strings.TrimSpace(config.BucketName)); err != nil {
		return nil, err
	}

	return &OssFileMigrator{
		service: &ossFileService{
			client:     client,
			bucketName: strings.TrimSpace(config.BucketName),
		},
	}, nil
}

// UploadLocalFile uploads an existing local file to the requested OSS object
// key and returns the provider path. The caller owns database updates.
func (m *OssFileMigrator) UploadLocalFile(ctx context.Context, sourcePath, objectKey, contentType string) (string, error) {
	if m == nil || m.service == nil {
		return "", fmt.Errorf("OSS file migrator is not initialized")
	}
	if err := utils.SafeObjectKey(objectKey); err != nil {
		return "", fmt.Errorf("invalid OSS object key: %w", err)
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("stat local source file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("local source is not a regular file")
	}

	src, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("open local source file: %w", err)
	}
	defer src.Close()

	if strings.TrimSpace(contentType) == "" {
		contentType = utils.GetContentTypeByExt(filepath.Ext(sourcePath))
	}
	if err := m.service.uploadReader(ctx, src, info.Size(), objectKey, contentType); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s/%s", ossScheme, m.service.bucketName, objectKey), nil
}

func (s *ossFileService) uploadReader(ctx context.Context, src io.Reader, size int64, objectName, contentType string) error {
	// Use Uploader for files > 10MB (auto multipart with concurrent uploads).
	const multipartThreshold = 10 * 1024 * 1024
	if size > multipartThreshold {
		uploader := s.client.NewUploader(func(uo *oss.UploaderOptions) {
			uo.PartSize = 10 * 1024 * 1024
			uo.ParallelNum = 3
		})
		_, err := uploader.UploadFrom(ctx, &oss.PutObjectRequest{
			Bucket:      oss.Ptr(s.bucketName),
			Key:         oss.Ptr(objectName),
			ContentType: oss.Ptr(contentType),
		}, src)
		if err != nil {
			return fmt.Errorf("failed to upload local file to OSS (multipart): %w", err)
		}
		return nil
	}

	_, err := s.client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:      oss.Ptr(s.bucketName),
		Key:         oss.Ptr(objectName),
		Body:        src,
		ContentType: oss.Ptr(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload local file to OSS: %w", err)
	}
	return nil
}

// CheckOssConnectivity tests OSS connectivity using the provided credentials.
func CheckOssConnectivity(ctx context.Context, endpoint, region, accessKey, secretKey, bucketName string) error {
	client, err := newOSSClient(endpoint, region, accessKey, secretKey)
	if err != nil {
		return err
	}

	exists, err := ossBucketExists(ctx, client, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check OSS bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist or is not accessible", bucketName)
	}
	if ossBrowserPreviewCORSEnabled() {
		if err := ensureOSSBrowserPreviewCORS(ctx, client, bucketName, ossBrowserPreviewAllowedOrigins()); err != nil {
			return fmt.Errorf("OSS direct browser preview CORS check failed: %w", err)
		}
	}
	return nil
}

// parseOssFilePath extracts bucket and object key from: oss://{bucket}/{objectKey}
func parseOssFilePath(filePath string) (bucketName string, objectKey string, err error) {
	if !strings.HasPrefix(filePath, ossScheme) {
		return "", "", fmt.Errorf("invalid OSS file path: %s", filePath)
	}

	rest := strings.TrimPrefix(filePath, ossScheme)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid OSS file path: %s", filePath)
	}
	return parts[0], parts[1], nil
}

// CheckConnectivity verifies OSS is reachable and the main bucket exists.
func (s *ossFileService) CheckConnectivity(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	exists, err := ossBucketExists(checkCtx, s.client, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check OSS bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", s.bucketName)
	}
	if ossBrowserPreviewCORSEnabled() {
		if err := ensureOSSBrowserPreviewCORS(checkCtx, s.client, s.bucketName, ossBrowserPreviewAllowedOrigins()); err != nil {
			return fmt.Errorf("OSS direct browser preview CORS check failed: %w", err)
		}
	}
	return nil
}

// SaveFile saves a file to OSS using the Uploader manager for large files.
func (s *ossFileService) SaveFile(ctx context.Context,
	file *multipart.FileHeader, tenantID uint64, knowledgeID string,
) (string, error) {
	ext := filepath.Ext(file.Filename)
	objectName := fmt.Sprintf("%s%d/%s/%s%s", s.pathPrefix, tenantID, knowledgeID, uuid.New().String(), ext)

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = utils.GetContentTypeByExt(ext)
	}

	// Use Uploader for files > 10MB (auto multipart with concurrent uploads)
	const multipartThreshold = 10 * 1024 * 1024
	if file.Size > multipartThreshold {
		uploader := s.client.NewUploader(func(uo *oss.UploaderOptions) {
			uo.PartSize = 10 * 1024 * 1024 // 10MB per part
			uo.ParallelNum = 3             // 3 concurrent uploads
		})

		_, err = uploader.UploadFrom(ctx,
			&oss.PutObjectRequest{
				Bucket:      oss.Ptr(s.bucketName),
				Key:         oss.Ptr(objectName),
				ContentType: oss.Ptr(contentType),
			},
			src,
		)
		if err != nil {
			return "", fmt.Errorf("failed to upload file to OSS (multipart): %w", err)
		}
	} else {
		_, err = s.client.PutObject(ctx, &oss.PutObjectRequest{
			Bucket:      oss.Ptr(s.bucketName),
			Key:         oss.Ptr(objectName),
			Body:        src,
			ContentType: oss.Ptr(contentType),
		})
		if err != nil {
			return "", fmt.Errorf("failed to upload file to OSS: %w", err)
		}
	}

	return fmt.Sprintf("oss://%s/%s", s.bucketName, objectName), nil
}

// SaveBytes saves bytes data to OSS.
// If temp is true and temp bucket is configured, saves to temp bucket.
// Otherwise saves to main bucket.
func (s *ossFileService) SaveBytes(ctx context.Context, data []byte, tenantID uint64, fileName string, temp bool) (string, error) {
	safeName, err := utils.SafeFileName(fileName)
	if err != nil {
		return "", fmt.Errorf("invalid file name: %w", err)
	}
	ext := filepath.Ext(safeName)

	targetBucket := s.bucketName
	client := s.client
	objectName := fmt.Sprintf("%s%d/exports/%s%s", s.pathPrefix, tenantID, uuid.New().String(), ext)

	if temp && s.tempClient != nil {
		targetBucket = s.tempBucketName
		client = s.tempClient
		objectName = fmt.Sprintf("exports/%d/%s%s", tenantID, uuid.New().String(), ext)
	}

	_, err = client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket:      oss.Ptr(targetBucket),
		Key:         oss.Ptr(objectName),
		Body:        bytes.NewReader(data),
		ContentType: oss.Ptr(utils.GetContentTypeByExt(ext)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload bytes to OSS: %w", err)
	}

	return fmt.Sprintf("oss://%s/%s", targetBucket, objectName), nil
}

// CopyFile copies an existing OSS object to a new knowledge-owned object using a
// server-side CopyObject (no data leaves OSS). The destination uses the same
// layout as SaveFile. Returns ErrCrossBackendCopy when srcPath is not an oss:// path.
func (s *ossFileService) CopyFile(ctx context.Context,
	srcPath string, tenantID uint64, knowledgeID string,
) (string, error) {
	srcBucket, srcKey, err := parseOssFilePath(srcPath)
	if err != nil {
		return "", fmt.Errorf("oss copy rejected source %q: %w", srcPath, ErrCrossBackendCopy)
	}
	if err := utils.SafeObjectKey(srcKey); err != nil {
		return "", fmt.Errorf("invalid source path: %w", err)
	}

	ext := filepath.Ext(srcPath)
	destKey := fmt.Sprintf("%s%d/%s/%s%s", s.pathPrefix, tenantID, knowledgeID, uuid.New().String(), ext)

	_, err = s.client.CopyObject(ctx, &oss.CopyObjectRequest{
		Bucket:       oss.Ptr(s.bucketName),
		Key:          oss.Ptr(destKey),
		SourceBucket: oss.Ptr(srcBucket),
		SourceKey:    oss.Ptr(srcKey),
	})
	if err != nil {
		return "", fmt.Errorf("failed to copy file in OSS: %w", err)
	}

	newPath := fmt.Sprintf("oss://%s/%s", s.bucketName, destKey)
	logger.Infof(ctx, "Copied OSS object %s to %s", srcPath, newPath)
	return newPath, nil
}

// GetFile retrieves a file from OSS by its path.
func (s *ossFileService) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	bucketName, objectName, err := parseOssFilePath(filePath)
	if err != nil {
		return nil, err
	}
	if err := utils.SafeObjectKey(objectName); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	var client *oss.Client
	if bucketName == s.tempBucketName && s.tempClient != nil {
		client = s.tempClient
	} else {
		client = s.client
	}

	resp, err := client.GetObject(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from OSS: %w", err)
	}

	return resp.Body, nil
}

// DeleteFile removes a file from OSS.
func (s *ossFileService) DeleteFile(ctx context.Context, filePath string) error {
	bucketName, objectName, err := parseOssFilePath(filePath)
	if err != nil {
		return err
	}
	if err := utils.SafeObjectKey(objectName); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	var client *oss.Client
	if bucketName == s.tempBucketName && s.tempClient != nil {
		client = s.tempClient
	} else {
		client = s.client
	}

	_, err = client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from OSS: %w", err)
	}

	return nil
}

// GetFileURL returns a presigned download URL for the file.
func (s *ossFileService) GetFileURL(ctx context.Context, filePath string) (string, error) {
	bucketName, objectName, err := parseOssFilePath(filePath)
	if err != nil {
		return "", err
	}
	if err := utils.SafeObjectKey(objectName); err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	// Bucket management and CORS checks always use the official storage client.
	var storageClient *oss.Client
	if bucketName == s.tempBucketName && s.tempClient != nil {
		storageClient = s.tempClient
	} else {
		storageClient = s.client
	}

	if ossBrowserPreviewCORSEnabled() {
		if err := ensureOSSBrowserPreviewCORSOnce(ctx, storageClient, bucketName); err != nil {
			logger.Warnf(
				ctx,
				"Failed to configure OSS CORS for direct browser preview: bucket=%s err=%v",
				bucketName,
				err,
			)
		}
	}

	signingClient := storageClient
	if bucketName == s.bucketName && s.publicClient != nil {
		signingClient = s.publicClient
	}

	// Generate presigned URL (valid for 24 hours)
	result, err := signingClient.Presign(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucketName),
		Key:    oss.Ptr(objectName),
	}, oss.PresignExpires(24*time.Hour))
	if err != nil {
		return "", fmt.Errorf("failed to generate OSS presigned URL: %w", err)
	}

	return result.URL, nil
}
