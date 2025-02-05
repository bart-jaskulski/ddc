package main

import (
	"encoding/json"
	"net/http"
)

type Documentation struct {
	Slug        string `json:"slug"`
	Mtime       int64  `json:"mtime"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Release     string `json:"release"`
	Description string `json:"description"`

  entries []DocEntry
}

func ListDocumentations() ([]Documentation, error) {
	resp, err := http.Get("https://devdocs.io/docs.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var docsets []Documentation
	if err := json.NewDecoder(resp.Body).Decode(&docsets); err != nil {
		return nil, err
	}
	return docsets, nil
}
