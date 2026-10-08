package file

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

func TestParseOssFilePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantBucket  string
		wantKey     string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid path with nested key",
			input:      "oss://my-bucket/123/exports/abc123.csv",
			wantBucket: "my-bucket",
			wantKey:    "123/exports/abc123.csv",
		},
		{
			name:       "valid path with simple key",
			input:      "oss://test-bucket/key",
			wantBucket: "test-bucket",
			wantKey:    "key",
		},
		{
			name:       "valid path with deep nesting",
			input:      "oss://bucket/prefix/tenant/exports/uuid.png",
			wantBucket: "bucket",
			wantKey:    "prefix/tenant/exports/uuid.png",
		},
		{
			name:        "invalid scheme",
			input:       "s3://bucket/key",
			wantErr:     true,
			errContains: "invalid OSS file path",
		},
		{
			name:        "empty path",
			input:       "",
			wantErr:     true,
			errContains: "invalid OSS file path",
		},
		{
			name:        "bucket only no key",
			input:       "oss://bucket/",
			wantErr:     true,
			errContains: "invalid OSS file path",
		},
		{
			name:        "scheme only",
			input:       "oss://",
			wantErr:     true,
			errContains: "invalid OSS file path",
		},
		{
			name:        "no slash after bucket",
			input:       "oss://bucket",
			wantErr:     true,
			errContains: "invalid OSS file path",
		},
		{
			name:        "empty bucket name",
			input:       "oss:///some-key",
			wantErr:     true,
			errContains: "invalid OSS file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := parseOssFilePath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseOssFilePath(%q) expected error, got bucket=%q key=%q", tt.input, bucket, key)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseOssFilePath(%q) error = %v, want containing %q", tt.input, err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("parseOssFilePath(%q) unexpected error: %v", tt.input, err)
				return
			}
			if bucket != tt.wantBucket {
				t.Errorf("parseOssFilePath(%q) bucket = %q, want %q", tt.input, bucket, tt.wantBucket)
			}
			if key != tt.wantKey {
				t.Errorf("parseOssFilePath(%q) key = %q, want %q", tt.input, key, tt.wantKey)
			}
		})
	}
}

func TestNewOSSClient(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		region    string
		accessKey string
		secretKey string
		wantErr   bool
	}{
		{
			name:      "valid parameters create client",
			endpoint:  "https://oss-cn-hangzhou.aliyuncs.com",
			region:    "cn-hangzhou",
			accessKey: "test-access-key",
			secretKey: "test-secret-key",
			wantErr:   false,
		},
		{
			name:      "custom endpoint",
			endpoint:  "https://custom-oss-endpoint.com",
			region:    "cn-shanghai",
			accessKey: "ak",
			secretKey: "sk",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newOSSClient(tt.endpoint, tt.region, tt.accessKey, tt.secretKey)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("newOSSClient() unexpected error: %v", err)
				return
			}
			if client == nil {
				t.Error("expected non-nil client")
			}
		})
	}
}

func TestOSSPublicEndpointForBucket(t *testing.T) {
	t.Setenv("OSS_PUBLIC_ENDPOINT", "https://knowledge.example.com")
	t.Setenv("OSS_BUCKET_NAME", "main-bucket")
	t.Setenv("OSS_PUBLIC_BUCKET_NAME", "")

	if got := ossPublicEndpointForBucket("main-bucket"); got != "https://knowledge.example.com" {
		t.Fatalf("ossPublicEndpointForBucket(main-bucket) = %q", got)
	}
	if got := ossPublicEndpointForBucket("other-bucket"); got != "" {
		t.Fatalf("ossPublicEndpointForBucket(other-bucket) = %q, want empty", got)
	}

	t.Setenv("OSS_PUBLIC_BUCKET_NAME", "other-bucket")
	if got := ossPublicEndpointForBucket("other-bucket"); got != "https://knowledge.example.com" {
		t.Fatalf("explicit public bucket endpoint = %q", got)
	}
}

func TestGetFileURLUsesPublicCNAMEClient(t *testing.T) {
	t.Setenv("OSS_BROWSER_PREVIEW_CORS_ENABLED", "false")

	storageClient, err := newOSSClient(
		"https://oss-cn-beijing.aliyuncs.com",
		"cn-beijing",
		"test-access-key",
		"test-secret-key",
	)
	if err != nil {
		t.Fatalf("newOSSClient() error = %v", err)
	}
	publicClient, err := newOSSCNAMEClient(
		"https://knowledge.example.com",
		"cn-beijing",
		"test-access-key",
		"test-secret-key",
	)
	if err != nil {
		t.Fatalf("newOSSCNAMEClient() error = %v", err)
	}

	service := &ossFileService{
		client:       storageClient,
		publicClient: publicClient,
		bucketName:   "main-bucket",
	}
	signedURL, err := service.GetFileURL(
		context.Background(),
		"oss://main-bucket/weknora/10000/file.xlsx",
	)
	if err != nil {
		t.Fatalf("GetFileURL() error = %v", err)
	}

	parsed, err := url.Parse(signedURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Host != "knowledge.example.com" {
		t.Fatalf("signed URL host = %q, want knowledge.example.com", parsed.Host)
	}
	if parsed.Path != "/weknora/10000/file.xlsx" {
		t.Fatalf("signed URL path = %q", parsed.Path)
	}
	if parsed.Query().Get("x-oss-signature") == "" {
		t.Fatal("signed URL is missing x-oss-signature")
	}
}

func TestCheckOssConnectivity_InvalidEndpoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should fail with an invalid/unreachable endpoint
	err := CheckOssConnectivity(ctx,
		"https://invalid-oss-endpoint-that-does-not-exist.local",
		"cn-hangzhou",
		"invalid-access-key",
		"invalid-secret-key",
		"nonexistent-bucket",
	)

	if err == nil {
		t.Error("CheckOssConnectivity with invalid endpoint should return an error")
	}
}

func TestOssEnsureBucket_NonExistent(t *testing.T) {
	client, err := newOSSClient(
		"https://oss-cn-hangzhou.aliyuncs.com",
		"cn-hangzhou",
		"test-invalid-key",
		"test-invalid-secret",
	)
	if err != nil {
		t.Fatalf("newOSSClient() error: %v", err)
	}

	// Bucket that definitely doesn't exist - should return error
	err = ossEnsureBucket(client, "this-bucket-definitely-does-not-exist-12345")
	if err == nil {
		t.Error("ossEnsureBucket with non-existent bucket should return an error")
	}
}

func TestOssEnsureBucket_CreateFails(t *testing.T) {
	client, err := newOSSClient(
		"https://oss-cn-hangzhou.aliyuncs.com",
		"cn-hangzhou",
		"test-invalid-key",
		"test-invalid-secret",
	)
	if err != nil {
		t.Fatalf("newOSSClient() error: %v", err)
	}

	// Use a bucket that does not exist so IsBucketExist returns false and the
	// create path is exercised; with invalid credentials PutBucket then fails.
	// A common name like "test-bucket" already exists globally on OSS, which
	// would short-circuit at IsBucketExist and make this assertion flaky.
	err = ossEnsureBucket(client, "weknora-nonexistent-bucket-create-fails-12345")
	if err == nil {
		t.Error("ossEnsureBucket with invalid credentials should return an error")
	}
}

func TestOssBucketExists_RejectsAuthenticationErrors(t *testing.T) {
	tests := []struct {
		code             string
		shouldRejectAuth bool
	}{
		{code: "SignatureDoesNotMatch", shouldRejectAuth: true},
		{code: "InvalidAccessKeyId", shouldRejectAuth: true},
		{code: "InvalidSecurityToken", shouldRejectAuth: true},
		{code: "SecurityTokenExpired", shouldRejectAuth: true},
		{code: "AccessDenied", shouldRejectAuth: false},
		{code: "NoSuchBucket", shouldRejectAuth: false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := &oss.ServiceError{Code: tt.code}
			if got := ossIsAuthenticationError(err); got != tt.shouldRejectAuth {
				t.Errorf("ossIsAuthenticationError(%q) = %t, want %t", tt.code, got, tt.shouldRejectAuth)
			}
		})
	}
}

func TestAppendOSSBrowserPreviewCORSRule(t *testing.T) {
	t.Run("adds browser read rule while preserving existing rules", func(t *testing.T) {
		existing := &oss.CORSConfiguration{
			CORSRules: []oss.CORSRule{
				{
					AllowedOrigins: []string{"https://admin.example.com"},
					AllowedMethods: []string{"PUT"},
				},
			},
		}

		config, changed, err := appendOSSBrowserPreviewCORSRule(existing, []string{"*"})
		if err != nil {
			t.Fatalf("appendOSSBrowserPreviewCORSRule() error = %v", err)
		}
		if !changed {
			t.Fatal("appendOSSBrowserPreviewCORSRule() changed = false, want true")
		}
		if len(config.CORSRules) != 2 {
			t.Fatalf("CORS rule count = %d, want 2", len(config.CORSRules))
		}
		if got := config.CORSRules[0].AllowedMethods[0]; got != "PUT" {
			t.Fatalf("existing CORS rule was modified: method = %q", got)
		}
		added := config.CORSRules[1]
		if len(added.AllowedOrigins) != 1 || added.AllowedOrigins[0] != "*" {
			t.Fatalf("added origins = %#v, want [*]", added.AllowedOrigins)
		}
		if !ossCORSRuleAllowsBrowserRead(added, "https://app.example.com") {
			t.Fatal("added rule does not allow browser GET")
		}
	})

	t.Run("keeps an existing wildcard GET rule unchanged", func(t *testing.T) {
		existing := &oss.CORSConfiguration{
			CORSRules: []oss.CORSRule{
				{
					AllowedOrigins: []string{"*"},
					AllowedMethods: []string{"GET"},
				},
			},
		}

		config, changed, err := appendOSSBrowserPreviewCORSRule(existing, []string{"https://app.example.com"})
		if err != nil {
			t.Fatalf("appendOSSBrowserPreviewCORSRule() error = %v", err)
		}
		if changed {
			t.Fatal("appendOSSBrowserPreviewCORSRule() changed = true, want false")
		}
		if config != existing {
			t.Fatal("existing CORS configuration should be returned unchanged")
		}
	})

	t.Run("requires every configured origin", func(t *testing.T) {
		existing := &oss.CORSConfiguration{
			CORSRules: []oss.CORSRule{
				{
					AllowedOrigins: []string{"https://app.example.com"},
					AllowedMethods: []string{"GET"},
				},
			},
		}

		config, changed, err := appendOSSBrowserPreviewCORSRule(
			existing,
			[]string{"https://app.example.com", "https://admin.example.com"},
		)
		if err != nil {
			t.Fatalf("appendOSSBrowserPreviewCORSRule() error = %v", err)
		}
		if !changed || len(config.CORSRules) != 2 {
			t.Fatalf("changed = %t, rule count = %d; want true and 2", changed, len(config.CORSRules))
		}
	})

	t.Run("rejects buckets at the OSS rule limit", func(t *testing.T) {
		existing := &oss.CORSConfiguration{
			CORSRules: make([]oss.CORSRule, ossBrowserPreviewCORSMaxRules),
		}

		_, changed, err := appendOSSBrowserPreviewCORSRule(existing, []string{"*"})
		if err == nil {
			t.Fatal("appendOSSBrowserPreviewCORSRule() expected an error")
		}
		if changed {
			t.Fatal("appendOSSBrowserPreviewCORSRule() changed = true, want false")
		}
		if !strings.Contains(err.Error(), "already has 10 CORS rules") {
			t.Fatalf("error = %q, want rule limit detail", err)
		}
	})
}

func TestEnsureOSSBrowserPreviewCORS_PreservesExistingRules(t *testing.T) {
	var putBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["cors"]; !ok {
			t.Errorf("request query = %q, want cors subresource", r.URL.RawQuery)
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<CORSConfiguration>
  <CORSRule>
    <AllowedOrigin>https://admin.example.com</AllowedOrigin>
    <AllowedMethod>PUT</AllowedMethod>
  </CORSRule>
</CORSConfiguration>`)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read PUT body: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			putBody = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := oss.NewClient(
		oss.LoadDefaultConfig().
			WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test-key", "test-secret", "")).
			WithRegion("cn-test").
			WithEndpoint(server.URL).
			WithUsePathStyle(true),
	)

	err := ensureOSSBrowserPreviewCORS(
		context.Background(),
		client,
		"test-bucket",
		[]string{"https://app.example.com"},
	)
	if err != nil {
		t.Fatalf("ensureOSSBrowserPreviewCORS() error = %v", err)
	}
	for _, expected := range []string{
		"<AllowedOrigin>https://admin.example.com</AllowedOrigin>",
		"<AllowedMethod>PUT</AllowedMethod>",
		"<AllowedOrigin>https://app.example.com</AllowedOrigin>",
		"<AllowedMethod>GET</AllowedMethod>",
		"<AllowedMethod>HEAD</AllowedMethod>",
	} {
		if !strings.Contains(putBody, expected) {
			t.Fatalf("PUT body missing %q: %s", expected, putBody)
		}
	}
}
