package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/sahilm/fuzzy"
)

type searchResult struct {
	docset  string
	entry   DocumentEntry
	matches []int // Positions of matches in the name
}

func (s searchResult) FilterValue() string { return s.entry.Name }

type searchDelegate struct{}

func (d searchDelegate) Height() int                             { return 1 }
func (d searchDelegate) Spacing() int                            { return 0 }
func (d searchDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d searchDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(searchResult)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s: %s", i.docset, i.entry.Name)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = selectedItemStyle.Render
	}

	fmt.Fprint(w, fn(str))
}

type SearchModel struct {
	list   list.Model
	cache  *Cache
	query  string
	docset string // Optional docset to search within
	err    error
}

// NewSearchModel creates a search model that searches across all documentations
// or within a specific docset if specified
func NewSearchModel(cache *Cache, query string, docset ...string) (SearchModel, error) {
	var specificDocset string
	if len(docset) > 0 && docset[0] != "" {
		specificDocset = docset[0]
	}

	var results []list.Item

	// If a specific docset is provided, only search within that docset
	if specificDocset != "" {
		// Check if the docset exists
		indexData, err := cache.GetIndex(specificDocset)
		if err != nil {
			return SearchModel{}, fmt.Errorf("documentation %s is not installed: %w", specificDocset, err)
		}

		var index struct {
			Entries []DocumentEntry `json:"entries"`
		}
		if err := json.Unmarshal(indexData, &index); err != nil {
			return SearchModel{}, fmt.Errorf("failed to parse index for %s: %w", specificDocset, err)
		}

		// Create a slice of strings for fuzzy matching
		names := make([]string, len(index.Entries))
		for i, entry := range index.Entries {
			names[i] = entry.Name
		}

		// Perform fuzzy search
		matches := fuzzy.Find(query, names)
		for _, match := range matches {
			results = append(results, searchResult{
				docset:  specificDocset,
				entry:   index.Entries[match.Index],
				matches: match.MatchedIndexes,
			})
		}
	} else {
		// Search across all installed documentations
		entries, err := os.ReadDir(cache.BaseDir)
		if err != nil {
			return SearchModel{}, err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			slug := entry.Name()
			indexData, err := cache.GetIndex(slug)
			if err != nil {
				continue
			}

			var index struct {
				Entries []DocumentEntry `json:"entries"`
			}
			if err := json.Unmarshal(indexData, &index); err != nil {
				continue
			}

			// Create a slice of strings for fuzzy matching
			names := make([]string, len(index.Entries))
			for i, entry := range index.Entries {
				names[i] = entry.Name
			}

			// Perform fuzzy search
			matches := fuzzy.Find(query, names)
			for _, match := range matches {
				results = append(results, searchResult{
					docset:  slug,
					entry:   index.Entries[match.Index],
					matches: match.MatchedIndexes,
				})
			}
		}
	}

	title := fmt.Sprintf("Search results for '%s'", query)
	if specificDocset != "" {
		title += fmt.Sprintf(" in %s", specificDocset)
	}

	l := list.New(results, searchDelegate{}, 80, 30)
	l.Title = title
	l.SetShowStatusBar(true)

	return SearchModel{
		list:   l,
		cache:  cache,
		query:  query,
		docset: specificDocset,
	}, nil
}

func (m SearchModel) Init() (tea.Model, tea.Cmd) {
	return m, nil
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "o":
			if m.list.SettingFilter() {
				break
			}
			if i, ok := m.list.SelectedItem().(searchResult); ok {
				// Open the selected entry with lynx
				htmlPath := filepath.Join(m.cache.GetHTMLDir(i.docset), strings.ReplaceAll(i.entry.Path, ".", "/")+".html")
				cmd := exec.Command("lynx", htmlPath)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					m.err = fmt.Errorf("failed to open documentation: %w, trying %s", err, htmlPath)
					return m, nil
				}
				return m, tea.Quit
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SearchModel) View() string {
	view := "\n" + m.list.View()
	if m.err != nil {
		view += "\nError: " + m.err.Error()
	}
	return view
}
