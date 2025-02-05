package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"io"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
	list     list.Model
	cache    *Cache
	query    string
	err      error
	quitting bool
}

func NewSearchModel(cache *Cache, query string) (SearchModel, error) {
	// Search across all installed documentations
	var results []list.Item
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

	l := list.New(results, searchDelegate{}, 80, 20)
	l.Title = fmt.Sprintf("Search results for '%s'", query)
	l.SetShowStatusBar(true)

	return SearchModel{
		list:  l,
		cache: cache,
		query: query,
	}, nil
}

func (m SearchModel) Init() tea.Cmd {
	return nil
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
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
	if m.quitting {
		return ""
	}

	view := "\n" + m.list.View()
	if m.err != nil {
		view += "\nError: " + m.err.Error()
	}
	return view
}

