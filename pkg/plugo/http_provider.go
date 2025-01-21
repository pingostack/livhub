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
	"time"
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

	return &HTTPProvider{
		url:          urlStr,
		format:       format,
		localPath:    localPath,
		httpClient:   &http.Client{Timeout: time.Second * 10},
		options:      options,
		fileProvider: fileProvider,
	}, nil
}

// downloadConfig downloads the remote configuration to local file
func (p *HTTPProvider) downloadConfig(ctx context.Context) error {
	// Check if local file exists before attempting download
	if _, err := os.Stat(p.localPath); err == nil {
		// Local file exists, try download but fallback to existing file if fails
		if err := p.tryDownload(ctx); err != nil {
			fmt.Printf("Warning: failed to download config from %s: %v, using existing file\n", p, err)
			return nil
		}
	} else {
		// No local file, must download successfully
		if err := p.tryDownload(ctx); err != nil {
			return fmt.Errorf("failed to download config and no local file exists: %w", err)
		}
	}
	return nil
}

// tryDownload attempts to download the remote config
func (p *HTTPProvider) tryDownload(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch config, status code: %d", resp.StatusCode)
	}

	// Create a temporary file in the same directory
	tmpFile := p.localPath + ".tmp"
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

	// Copy response body to temporary file
	if _, err = io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to write config to temporary file: %w", err)
	}

	// Ensure all data is written to disk
	if err = f.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close the file before renaming
	if err = f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomically replace the old file with the new one
	if err = os.Rename(tmpFile, p.localPath); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	return nil
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

// Close cleans up temporary files
func (p *HTTPProvider) Close() error {
	//return os.Remove(p.localPath)

	return nil
}

func (p *HTTPProvider) LocalPath() string {
	return p.localPath
}
