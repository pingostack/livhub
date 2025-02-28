package plugo

import (
	"fmt"
	"path/filepath"
)

// FileProvider implements ConfigProvider for local files
type FileProvider struct {
	path   string
	format string
}

// NewFileProvider creates a new file-based configuration provider
func NewFileProvider(path string) (*FileProvider, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %s: %w", path, err)
	}

	// Get format without the leading dot
	format := filepath.Ext(absPath)
	if format != "" {
		format = format[1:] // Remove the leading dot
	}

	return &FileProvider{
		path:   absPath,
		format: format,
	}, nil
}

// String returns a string representation of the provider
func (p *FileProvider) String() string {
	return fmt.Sprintf("file://%s", p.path)
}

func (p *FileProvider) LocalPath() string {
	return p.path
}

func (p *FileProvider) Start() error {
	return nil
}

func (p *FileProvider) Close() error {
	return nil
}
