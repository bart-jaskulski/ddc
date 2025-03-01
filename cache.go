package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const DefaultDevDocsDir = ".local/share/devdocs"

type DocMeta struct {
	Release string `json:"release"`
	Version string `json:"version"`
	Mtime   int64  `json:"mtime"`
}

type Cache struct {
	BaseDir string
}

func newCache() *Cache {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, DefaultDevDocsDir)
	return &Cache{BaseDir: baseDir}
}

func (c *Cache) EnsureDir(slug string) error {
	return os.MkdirAll(filepath.Join(c.BaseDir, slug), 0755)
}

func (c *Cache) GetDocPath(slug string) string {
	return filepath.Join(c.BaseDir, slug)
}

func (c *Cache) SaveMeta(slug string, meta DocMeta) error {
	path := filepath.Join(c.BaseDir, slug, "meta.json")
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Cache) GetMeta(slug string) (DocMeta, error) {
	path := filepath.Join(c.BaseDir, slug, "meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return DocMeta{}, err
	}

	var meta DocMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return DocMeta{}, err
	}
	return meta, nil
}

func (c *Cache) GetIndex(slug string) ([]byte, error) {
	path := filepath.Join(c.BaseDir, slug, "index.json")
	return os.ReadFile(path)
}

func (c *Cache) GetDB(slug string) ([]byte, error) {
	path := filepath.Join(c.BaseDir, slug, "db.json")
	return os.ReadFile(path)
}

func (c *Cache) DocsetExists(slug string) bool {
	path := filepath.Join(c.BaseDir, slug)
	_, err := os.Stat(path)
	return err == nil
}

func (c *Cache) GetHTMLDir(slug string) string {
	return filepath.Join(c.BaseDir, slug, "html")
}

func (c *Cache) EnsureHTMLDir(slug string) error {
	return os.MkdirAll(c.GetHTMLDir(slug), 0755)
}

// fixRelativeLinks adds .html extension to relative links in HTML content
func (c *Cache) fixRelativeLinks(content string) string {
	re := regexp.MustCompile(`href="([^"]*)"`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the URL from href="url"
		url := match[6 : len(match)-1]

		// Skip if it's an absolute URL or already has .html extension
		if strings.Contains(url, "://") || // Has protocol (http://, https://, etc)
			strings.HasPrefix(url, "//") || // Protocol-relative URL
			strings.HasPrefix(url, "mailto:") || // Email link
			strings.HasSuffix(url, ".html") { // Already has .html
			return match
		}

		// Remove any trailing slash
		url = strings.TrimSuffix(url, "/")

		// Add .html extension
		return fmt.Sprintf(`href="%s.html"`, url)
	})
}

func (c *Cache) SaveHTML(slug string, path string, content string) error {
	// Convert dot notation to filesystem path
	parts := strings.Split(path, ".")
	htmlPath := filepath.Join(c.GetHTMLDir(slug), filepath.Join(parts...))

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0755); err != nil {
		return fmt.Errorf("failed to create HTML directory: %w", err)
	}

	// Add .html extension if not present
	if !strings.HasSuffix(htmlPath, ".html") {
		htmlPath += ".html"
	}

	// Fix relative links in content
	fixedContent := c.fixRelativeLinks(content)

	return os.WriteFile(htmlPath, []byte(fixedContent), 0644)
}
