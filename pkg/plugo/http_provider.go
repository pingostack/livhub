package plugo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"encoding/json"

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
}

// HTTPProvider implements ConfigProvider for HTTP/HTTPS URLs by downloading to local file
type HTTPProvider struct {
	url          string
	format       string
	localPath    string
	httpClient   *http.Client
	options      *RemoteConfigOptions
	fileProvider *FileProvider
	ctx          context.Context
	cancel       context.CancelFunc
	onUpdate     func()
	writeMu      sync.Mutex // Protect file writes
}

var DefaultRemoteConfigOptions = &RemoteConfigOptions{
	PollInterval: defaultPollInterval,
	MaxRetries:   maxRetries,
}

// NewHTTPProvider creates a new HTTP-based configuration provider
func NewHTTPProvider(urlStr string, format string, options *RemoteConfigOptions) (*HTTPProvider, error) {
	_, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", urlStr, err)
	}

	if options == nil {
		options = DefaultRemoteConfigOptions
	}

	// Create temp directory for downloaded configs if it doesn't exist
	tempDir := filepath.Join(os.TempDir(), "plugo", "configs")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create a unique and safe filename using URL hash
	hasher := sha256.New()
	hasher.Write([]byte(urlStr))
	fileName := fmt.Sprintf("%s.%s", hex.EncodeToString(hasher.Sum(nil))[:32], format)
	localPath := filepath.Join(tempDir, fileName)

	// Create file provider for the local file
	fileProvider, err := NewFileProvider(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file provider: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	provider := &HTTPProvider{
		url:          urlStr,
		format:       format,
		localPath:    localPath,
		httpClient:   &http.Client{Timeout: time.Second * 10},
		options:      options,
		fileProvider: fileProvider,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start polling if interval is set
	if options.PollInterval > 0 {
		go provider.startPolling()
	}

	return provider, nil
}

// startPolling starts a goroutine to periodically poll for config updates
func (p *HTTPProvider) startPolling() {
	ticker := time.NewTicker(p.options.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.downloadConfig(p.ctx); err != nil {
				fmt.Printf("Warning: failed to poll config from %s: %v\n", p, err)
			}
		}
	}
}

// downloadConfig downloads the remote configuration to local file
func (p *HTTPProvider) downloadConfig(ctx context.Context) error {
	// Check if local file exists before attempting download
	if _, err := os.Stat(p.localPath); err == nil {
		// Local file exists, try download but fallback to existing file if fails
		if err := p.tryDownload(); err != nil {
			fmt.Printf("Warning: failed to download config from %s: %v, using existing file\n", p, err)
			return nil
		}
	} else {
		// No local file, must download successfully
		if err := p.tryDownload(); err != nil {
			return fmt.Errorf("failed to download config and no local file exists: %w", err)
		}
	}
	return nil
}

// tryDownload attempts to download the remote config
func (p *HTTPProvider) tryDownload() error {
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

	// Generate file paths
	basePath := strings.TrimSuffix(p.localPath, filepath.Ext(p.localPath))
	tmpFile := basePath + ".tmp" + filepath.Ext(p.localPath)
	completeFile := basePath + ".complete" + filepath.Ext(p.localPath)

	// Create temporary file
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		f.Close()
		// Clean up temp file in case of failure
		if err != nil {
			os.Remove(tmpFile)
		}
	}()

	// Write the entire content at once
	if _, err = f.Write(body); err != nil {
		return fmt.Errorf("failed to write config to temporary file: %w", err)
	}

	// Ensure all data is written to disk
	if err = f.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close the file before further operations
	if err = f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Rename tmp to complete
	if err = os.Rename(tmpFile, completeFile); err != nil {
		return fmt.Errorf("failed to rename tmp to complete: %w", err)
	}

	// Remove old symlink if it exists
	os.Remove(p.localPath)

	// Create new symlink pointing to complete file
	if err = os.Symlink(filepath.Base(completeFile), p.localPath); err != nil {
		// If symlink fails, clean up the complete file
		os.Remove(completeFile)
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Clean up old complete files
	pattern := basePath + ".complete*"
	oldFiles, _ := filepath.Glob(pattern)
	for _, f := range oldFiles {
		// Skip the current complete file
		if f == completeFile {
			continue
		}
		os.Remove(f)
	}

	return nil
}

// isValidConfig checks if the config content is valid
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

// Read downloads the remote config and delegates to FileProvider
func (p *HTTPProvider) Read(ctx context.Context) (io.Reader, string, error) {
	if err := p.downloadConfig(ctx); err != nil {
		return nil, "", err
	}
	return p.fileProvider.Read(ctx)
}

// String returns a string representation of the provider
func (p *HTTPProvider) String() string {
	return p.url
}

// Close cleans up temporary files and stops polling
func (p *HTTPProvider) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	//return os.Remove(p.localPath)
	return nil
}

func (p *HTTPProvider) LocalPath() string {
	return p.localPath
}
