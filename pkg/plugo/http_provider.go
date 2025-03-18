package plugo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPollInterval = time.Minute
	maxRetries          = 3
)

// RemoteConfigOptions contains options for remote configuration providers
type RemoteConfigOptions struct {
	// PollInterval is the interval between config checks
	PollInterval time.Duration
	// MaxRetries is the maximum number of retries on failure
	MaxRetries int
	// Format is the format of the configuration file
	Format string
	// TmpDir is the temporary directory for downloading files
	TempDir string
}

// HTTPProvider implements ConfigProvider for HTTP URLs
type HTTPProvider struct {
	url          string
	format       string
	localPath    string
	httpClient   *http.Client
	options      *RemoteConfigOptions
	fileProvider *FileProvider
	ctx          context.Context
	cancel       context.CancelFunc
}

var DefaultRemoteConfigOptions = &RemoteConfigOptions{
	PollInterval: defaultPollInterval,
	MaxRetries:   maxRetries,
}

// NewHTTPProvider creates a new HTTP provider
func NewHTTPProvider(ctx context.Context, urlStr string, options *RemoteConfigOptions) (*HTTPProvider, error) {
	if options == nil {
		options = DefaultRemoteConfigOptions
	}

	// Parse URL and validate scheme
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// Create temp directory if it doesn't exist
	if options.TempDir == "" {
		options.TempDir = filepath.Join(os.TempDir(), "plugo", "configs")
	}
	tempDir := options.TempDir
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate a unique filename based on URL hash
	format := options.Format
	if format == "" {
		format = "json"
	}
	hasher := sha256.New()
	hasher.Write([]byte(urlStr))
	fileName := fmt.Sprintf("%s.%s", hex.EncodeToString(hasher.Sum(nil))[:32], format)
	localPath := filepath.Join(tempDir, fileName)

	// Create file provider for the local file
	fileProvider, err := NewFileProvider(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file provider: %w", err)
	}

	provider := &HTTPProvider{
		url:          urlStr,
		format:       format,
		localPath:    localPath,
		httpClient:   &http.Client{Timeout: time.Second * 10},
		options:      options,
		fileProvider: fileProvider,
	}

	ctx, cancel := context.WithCancel(ctx)
	provider.ctx = ctx
	provider.cancel = cancel

	return provider, nil
}

func (p *HTTPProvider) Start() error {

	// Do initial download
	if err := p.downloadConfig(); err != nil {
		return fmt.Errorf("failed initial download: %w", err)
	}

	// Start polling if interval is set
	if p.options.PollInterval > 0 {
		go p.startPolling()
	}

	return nil
}

func (p *HTTPProvider) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// String returns a string representation of the provider
func (p *HTTPProvider) String() string {
	return p.url
}

// LocalPath returns the path to the local configuration file
func (p *HTTPProvider) LocalPath() string {
	return p.localPath
}

func (p *HTTPProvider) startPolling() {
	ticker := time.NewTicker(p.options.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.downloadConfig(); err != nil {
				fmt.Printf("Warning: failed to poll config from %s: %v\n", p, err)
			}
		}
	}
}

func (p *HTTPProvider) downloadConfig() error {
	// Get response body first
	resp, err := p.httpClient.Get(p.url)
	if err != nil {
		return fmt.Errorf("failed to download config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download config: status %d", resp.StatusCode)
	}

	// Read entire response into memory to validate it
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Validate response body is valid config
	if !p.isValidConfig(body) {
		return fmt.Errorf("invalid config content")
	}

	// Check if content has changed
	currentContent, err := os.ReadFile(p.localPath)
	if err == nil && bytes.Equal(currentContent, body) {
		// Content hasn't changed, no need to update
		return nil
	}

	// Generate temporary file path
	tmpFile := p.localPath + ".tmp"

	// Write to temporary file
	if err = os.WriteFile(tmpFile, body, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	defer func() {
		// Clean up temp file in case of error
		if _, err := os.Stat(tmpFile); err == nil {
			os.Remove(tmpFile)
		}
	}()

	// Rename temp file to target file
	if err = os.Rename(tmpFile, p.localPath); err != nil {
		return fmt.Errorf("failed to rename temporary file to target file: %w", err)
	}

	return nil
}

func (p *HTTPProvider) isValidConfig(content []byte) bool {
	// For JSON format
	if p.format == "json" {
		return json.Valid(content)
	}

	// For YAML format
	if p.format == "yaml" || p.format == "yml" {
		var out interface{}
		return yaml.Unmarshal(content, &out) == nil
	}

	return true // For other formats, assume valid
}
