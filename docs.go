package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type DevDoc struct {
	cache *Cache
}

func newDocs(cache *Cache) *DevDoc {
	return &DevDoc{cache: cache}
}

func (c *DevDoc) DownloadDocSet(docset Documentation) error {
	if err := c.cache.EnsureDir(docset.Slug); err != nil {
		return err
	}

	// Download index.json
	if err := c.downloadFile(
		fmt.Sprintf("https://devdocs.io/docs/%s/index.json?%d",
			docset.Slug, docset.Mtime),
		filepath.Join(c.cache.GetDocPath(docset.Slug), "index.json"),
	); err != nil {
		return err
	}

	// Download db.json
	if err := c.downloadFile(
		fmt.Sprintf("https://documents.devdocs.io/%s/db.json?%d",
			docset.Slug, docset.Mtime),
		filepath.Join(c.cache.GetDocPath(docset.Slug), "db.json"),
	); err != nil {
		return err
	}

	if err := c.cache.SaveMtime(docset.Slug, docset.Mtime); err != nil {
		return err
	}

	// Unpack documentation into HTML files
	return c.unpackHTML(docset.Slug)
}

func (c *DevDoc) downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (c *DevDoc) GetDocumentation(slug string) ([]DocEntry, error) {
	data, err := c.cache.GetIndex(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to read index.json: %w", err)
	}

	var index struct {
		Entries []DocEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index.json: %w", err)
	}

	return index.Entries, nil
}

func (c *DevDoc) GetDocument(slug, path string) (string, error) {
	data, err := c.cache.GetDB(slug)
	if err != nil {
		return "", fmt.Errorf("failed to read db.json: %w", err)
	}

	var docs map[string]string
	if err := json.Unmarshal(data, &docs); err != nil {
		return "", fmt.Errorf("failed to parse db.json: %w", err)
	}

	content, ok := docs[path]
	if !ok {
		return "", fmt.Errorf("document not found: %s", path)
	}

	return content, nil
}

func (c *DevDoc) IsDocSetInstalled(slug string) bool {
	return c.cache.DocsetExists(slug)
}

func (c *DevDoc) unpackHTML(slug string) error {
	// Ensure HTML directory exists
	if err := c.cache.EnsureHTMLDir(slug); err != nil {
		return fmt.Errorf("failed to create HTML directory: %w", err)
	}

	// Read db.json content
	data, err := c.cache.GetDB(slug)
	if err != nil {
		return fmt.Errorf("failed to read db.json: %w", err)
	}

	var docs map[string]string
	if err := json.Unmarshal(data, &docs); err != nil {
		return fmt.Errorf("failed to parse db.json: %w", err)
	}

	// Process each documentation entry
	for path, content := range docs {
		if err := c.cache.SaveHTML(slug, path, content); err != nil {
			return fmt.Errorf("failed to save HTML for %s: %w", path, err)
		}
	}

	return nil
}

func (c *DevDoc) NeedsUpdate(docset Documentation) (bool, error) {
	if !c.IsDocSetInstalled(docset.Slug) {
		return true, nil
	}

	currentMtime, err := c.cache.GetMtime(docset.Slug)
	if err != nil {
		return true, nil
	}

	return currentMtime < docset.Mtime, nil
}
