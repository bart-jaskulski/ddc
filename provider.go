package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type Documentation struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Type    string `json:"type"`
	Mtime   int64  `json:"mtime"`
	Version string `json:"version"`
	Release string `json:"release"`

	entries      []DocumentEntry
	versions     []Documentation
	showVersions bool
	isVersion    bool // Indicates if this is a version entry
}

type DocumentationVersion struct {
	Slug    string `json:"slug"`
	Version string `json:"version"`
	Release string `json:"release"`
	Mtime   int64  `json:"mtime"`
}

func (d Documentation) DisplayName() string {
	if d.isVersion {
		return fmt.Sprintf("  %s", d.Release)
	}
	return d.Name
}

func (d *Documentation) Kind() string {
	return strings.ToLower(d.Name)
}

func (d *Documentation) AddVersion(version Documentation) {
	d.versions = append(d.versions, version)
}

func (d *Documentation) ListVersions() []Documentation {
	if len(d.versions) > 1 {
		// Bubble sort with natural version comparison
		for i := 0; i < len(d.versions)-1; i++ {
			for j := 0; j < len(d.versions)-1-i; j++ {
				// Compare versions in reverse order (newest first)
				var cmp1, cmp2 string
				if d.versions[j].Version != "" {
					cmp1 = d.versions[j].Version
				} else {
					cmp1 = d.versions[j].Release
				}

				if d.versions[j+1].Version != "" {
					cmp2 = d.versions[j+1].Version
				} else {
					cmp2 = d.versions[j+1].Release
				}
				if CompareVersions(cmp1, cmp2) > 0 {
					d.versions[j], d.versions[j+1] = d.versions[j+1], d.versions[j]
				}
			}
		}
	}

	return d.versions
}

func (d *Documentation) GetLatestVersion() Documentation {
	return *d
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
	if !doc.isVersion {
		if m.Width() >= 40 {
			if d.cache.DocsetExists(doc.Kind()) {
				prefix = "[✓] "
			} else {
				prefix = "[ ] "
			}
		}
	}

	name := doc.DisplayName()
	if doc.showVersions && !doc.isVersion {
		name += " ▼" // Show dropdown indicator when versions are visible
	} else if len(doc.versions) > 0 && !doc.isVersion {
		name += " ▶" // Show there are hidden versions
	}

	var str string
	width := m.Width()
	if width >= 40 {
		str = fmt.Sprintf("%s%-25s", prefix, name)
	} else {
		str = fmt.Sprintf("%s%s", prefix, name)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(strings.Join(s, " "))
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

	l := list.New(items, delegate, 80, 30)
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

func (m ProviderModel) Init() (tea.Model, tea.Cmd) {
	return m, nil
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
		case "tab":
			if i, ok := m.list.SelectedItem().(Documentation); ok && !i.isVersion {
				items := m.list.Items()
				index := m.list.Index()

				// Toggle showVersions
				newDoc := i
				newDoc.showVersions = !newDoc.showVersions
				items[index] = newDoc

				if newDoc.showVersions {
					// Insert versions after this item
					// TODO: this is very bad, as we are sorting as a side effect and just then using internal
					// version field, but I don't want to think how to change it at the moment
					i.ListVersions()
					newItems := make([]list.Item, 0, len(items)+len(i.versions))
					newItems = append(newItems, items[:index+1]...)
					for idx := len(i.versions) - 1; idx >= 0; idx-- {
						v := i.versions[idx]
						v.isVersion = true
						newItems = append(newItems, v)
					}
					newItems = append(newItems, items[index+1:]...)
					m.list.SetItems(newItems)
				} else {
					// Remove version items
					newItems := make([]list.Item, 0, len(items))
					for _, item := range items {
						if doc, ok := item.(Documentation); ok {
							if !doc.isVersion {
								newItems = append(newItems, doc)
							}
						}
					}
					m.list.SetItems(newItems)
				}
				return m, nil
			}
		case "ctrl+c":
			return m, tea.Quit
		case "space":
			if i, ok := m.list.SelectedItem().(Documentation); ok {
				m.selected[i.Slug] = !m.selected[i.Slug]
			}
			return m, nil
		case "i":
			if m.list.SettingFilter() {
				break
			}
			if i, ok := m.list.SelectedItem().(Documentation); ok {
				if !m.cache.DocsetExists(i.Kind()) {
					docToDownload := i
					if !i.isVersion {
						// If not a version entry, get the latest version
						if len(i.versions) > 0 {
							docToDownload = i.GetLatestVersion()
						}
					}
					m.downloading = docToDownload.Slug
					return m, func() tea.Msg {
						err := m.client.DownloadDocSet(&docToDownload)
						return downloadMsg{slug: docToDownload.Slug, success: err == nil, err: err}
					}
				}
			}
		case "x":
			if m.list.SettingFilter() {
				break
			}
			if i, ok := m.list.SelectedItem().(Documentation); ok {
				if m.cache.DocsetExists(i.Kind()) {
					m.confirming = i.Kind()
					return m, nil
				}
			}
		case "y":
			if m.list.SettingFilter() {
				break
			}
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
			if m.list.SettingFilter() {
				break
			}
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

	var allDocs []Documentation
	if err := json.NewDecoder(resp.Body).Decode(&allDocs); err != nil {
		return nil, err
	}

	// Group by type
	docMap := make(map[string]*Documentation)
	for _, doc := range allDocs {
		if existing, ok := docMap[doc.Kind()]; ok {
			existing.AddVersion(doc)
		} else {
			// Add self to versions list
			doc.AddVersion(doc)
			docMap[doc.Kind()] = &doc
		}
	}

	// Convert map back to slice and sort by Name
	result := make([]Documentation, 0, len(docMap))
	for _, doc := range docMap {
		result = append(result, *doc)
	}

	// Sort by Name
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if strings.ToLower(result[i].Name) > strings.ToLower(result[j].Name) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}
