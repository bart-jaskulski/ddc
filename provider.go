package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Documentation struct {
	Slug        string `json:"slug"`
	Mtime       int64  `json:"mtime"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Release     string `json:"release"`
	Description string `json:"description"`

	entries []DocumentEntry
}

func (d Documentation) FilterValue() string { return d.Name }

type docDelegate struct {
	selected map[string]bool
	cache    *Cache
}

func (d docDelegate) Height() int                             { return 1 }
func (d docDelegate) Spacing() int                            { return 0 }
func (d docDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d docDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	doc, ok := listItem.(Documentation)
	if !ok {
		return
	}

	var prefix string
	if m.Width() >= 40 {
		if d.cache.DocsetExists(doc.Slug) {
			prefix = "[✓] "
		} else {
			prefix = "[ ] "
		}
	}

	name := doc.GetDisplayName()
	desc := doc.Description
	var str string

	width := m.Width()
	if width >= 40 {
		maxDescLen := width - len(name) - len(prefix) - 5
		if len(desc) > maxDescLen {
			desc = desc[:maxDescLen-3] + "..."
		}
		str = fmt.Sprintf("%s%-25s  %s", prefix, name, desc)
	} else {
		str = fmt.Sprintf("%s%s", prefix, name)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type downloadMsg struct {
	slug    string
	success bool
	err     error
}

type removeMsg struct {
	slug    string
	success bool
	err     error
}

type ProviderModel struct {
	list        list.Model
	selected    map[string]bool
	cache       *Cache
	client      *DevDoc
	downloading string // slug of doc being downloaded
	removing    string // slug of doc being removed
	confirming  string // slug of doc pending removal confirmation
}

func NewProviderModel(docsets []Documentation, cache *Cache, client *DevDoc) ProviderModel {
	items := make([]list.Item, len(docsets))
	for i, ds := range docsets {
		items[i] = ds
	}

	delegate := docDelegate{
		selected: make(map[string]bool),
		cache:    cache,
	}

	l := list.New(items, delegate, 80, 20)
	l.Title = "Available Documentation Sets"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return ProviderModel{
		list:     l,
		selected: delegate.selected,
		cache:    cache,
		client:   client,
	}
}

func (m ProviderModel) Init() tea.Cmd {
	return nil
}

func (m ProviderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadMsg:
		m.downloading = ""
		if !msg.success {
			// TODO: Show error message
			return m, tea.Quit
		}
		return m, nil

	case removeMsg:
		m.removing = ""
		if !msg.success {
			// TODO: Show error message
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if !m.list.SettingFilter() {
				return m, tea.Quit
			}
		case "ctrl+c":
			return m, tea.Quit
		case "space":
			if i, ok := m.list.SelectedItem().(Documentation); ok {
				m.selected[i.Slug] = !m.selected[i.Slug]
			}
			return m, nil
		case "enter":
			if i, ok := m.list.SelectedItem().(Documentation); ok {
				if !m.cache.DocsetExists(i.Slug) {
					m.downloading = i.Slug
					return m, func() tea.Msg {
						err := m.client.DownloadDocSet(i)
						return downloadMsg{slug: i.Slug, success: err == nil, err: err}
					}
				}
			}
		case "x":
			if i, ok := m.list.SelectedItem().(Documentation); ok {
				if m.cache.DocsetExists(i.Slug) {
					m.confirming = i.Slug
					return m, nil
				}
			}
		case "y":
			if m.confirming != "" {
				slug := m.confirming
				m.confirming = ""
				m.removing = slug
				return m, func() tea.Msg {
					err := os.RemoveAll(m.cache.GetDocPath(slug))
					return removeMsg{slug: slug, success: err == nil, err: err}
				}
			}
		case "n":
			if m.confirming != "" {
				m.confirming = ""
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ProviderModel) View() string {
	var status string
	if m.downloading != "" {
		status = fmt.Sprintf("\nDownloading %s...", m.downloading)
	}
	if m.removing != "" {
		status = fmt.Sprintf("\nRemoving %s...", m.removing)
	}
	if m.confirming != "" {
		status = fmt.Sprintf("\nAre you sure you want to remove %s? (y/n)", m.confirming)
	}

	view := "\n" + m.list.View()
	if status != "" {
		// Insert status before the last newline (where help text usually is)
		if idx := strings.LastIndex(view, "\n"); idx != -1 {
			view = view[:idx] + status + view[idx:]
		}
	}
	return view
}

func (m ProviderModel) GetSelected() []Documentation {
	var selected []Documentation
	for i := 0; i < len(m.list.Items()); i++ {
		if item, ok := m.list.Items()[i].(Documentation); ok {
			if m.selected[item.Slug] {
				selected = append(selected, item)
			}
		}
	}
	return selected
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
