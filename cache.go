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

// GetHTMLPath returns the path to an HTML file, correctly handling fragments
func (c *Cache) GetHTMLPath(slug string, path string) (string, string) {
	// Split path and fragment
	var fragment string
	if idx := strings.Index(path, "#"); idx >= 0 {
		fragment = path[idx:]
		path = path[:idx]
	}

	// Convert dot notation to filesystem path
	parts := strings.Split(path, ".")
	htmlPath := filepath.Join(c.GetHTMLDir(slug), filepath.Join(parts...))

	// Add .html extension if not present
	if !strings.HasSuffix(htmlPath, ".html") {
		htmlPath += ".html"
	}

	return htmlPath, fragment
}

func (c *Cache) EnsureHTMLDir(slug string) error {
	return os.MkdirAll(c.GetHTMLDir(slug), 0755)
}

func (c *Cache) SaveHTML(slug string, path string, content string) error {
	// Get the file path, stripping any fragment
	htmlPath, _ := c.GetHTMLPath(slug, path)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0755); err != nil {
		return fmt.Errorf("failed to create HTML directory: %w", err)
	}


	// Fix relative links in content - add the current directory info for relative path resolution
	currentDir := filepath.Dir(path)
	fmt.Println(htmlPath, path, currentDir)
	fixedContent := c.fixRelativeLinksWithContext(content, currentDir)

	return os.WriteFile(htmlPath, []byte(fixedContent), 0644)
}

// calculateRelativePath computes a relative path from source to target directory
func calculateRelativePath(sourceDirParts, targetDirParts []string) string {
	// Find common prefix length
	commonLength := 0
	for i := 0; i < len(sourceDirParts) && i < len(targetDirParts); i++ {
		if sourceDirParts[i] != targetDirParts[i] {
			break
		}
		commonLength++
	}
	
	// Build relative path
	var result strings.Builder
	
	// Add "../" for each directory level we need to go up
	for i := 0; i < len(sourceDirParts)-commonLength; i++ {
		if i > 0 {
			result.WriteString("/")
		}
		result.WriteString("..")
	}
	
	// Add path to target
	for i := commonLength; i < len(targetDirParts); i++ {
		if result.Len() > 0 {
			result.WriteString("/")
		}
		result.WriteString(targetDirParts[i])
	}
	
	// If empty, we're referring to the same directory
	if result.Len() == 0 {
		return "."
	}
	
	return result.String()
}

// fixRelativeLinksWithContext adds .html extension to relative links in HTML content
// with awareness of the current document's directory
func (c *Cache) fixRelativeLinksWithContext(content string, currentDir string) string {
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

		// Handle empty URLs or just fragments
		if url == "" || url == "." || url == "./" {
			return `href="index.html"`
		}
		
		// Handle URLs that are just fragments
		if strings.HasPrefix(url, "#") {
			return fmt.Sprintf(`href="index.html%s"`, url)
		}

		// Handle absolute paths within the docset
		isAbsolutePath := strings.HasPrefix(url, "/")
		if isAbsolutePath {
			// Remove the leading slash for processing
			url = strings.TrimPrefix(url, "/")
		}

		// Remove any trailing slash
		url = strings.TrimSuffix(url, "/")

		// Split URL and fragment identifier
		urlParts := strings.SplitN(url, "#", 2)
		baseUrl := urlParts[0]
		
		// If baseUrl is empty or just ".", use index.html
		if baseUrl == "" || baseUrl == "." {
			baseUrl = "index"
		}
		
		// Handle dotted paths (convert dots to directory separators)
		if strings.Contains(baseUrl, ".") && !strings.Contains(baseUrl, "/") {
			// This looks like a dotted path (like "language.types.array")
			// Convert dots to slashes
			targetPath := strings.ReplaceAll(baseUrl, ".", "/")
			
			// If we have a current directory context, create a proper relative path
			if currentDir != "" && currentDir != "." {
				// Split the paths into components
				targetParts := strings.Split(targetPath, "/")
				currentParts := strings.Split(currentDir, "/")
				fmt.Println(currentParts, targetParts)
				
				// Calculate the relative path (how many "../" we need)
				baseUrl = calculateRelativePath(currentParts, targetParts)
			} else {
				// No current directory, just use the slash path
				baseUrl = targetPath
			}
		} else if !isAbsolutePath && !strings.Contains(url, "..") && currentDir != "" && currentDir != "." {
			// For simple relative paths that aren't using dot notation and aren't already absolute
			if !strings.Contains(baseUrl, "/") {
				// Simple relative path in same directory
				baseUrl = filepath.Join(currentDir, baseUrl)
			}
		}
		
		// Add .html extension to the base URL
		if len(urlParts) > 1 {
			// Reassemble with fragment
			// Don't use the leading slash - causes file:// URL issues
			return fmt.Sprintf(`href="%s.html#%s"`, baseUrl, urlParts[1])
		}
		
		// No fragment
		return fmt.Sprintf(`href="%s.html"`, baseUrl)
	})
}
