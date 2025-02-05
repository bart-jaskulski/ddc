package main

import (
	"fmt"
	"io"
  "strings"
  "os/exec"
  "os"
  "path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type entryItem DocumentEntry

func (i entryItem) FilterValue() string { return i.Name }

type entryDelegate struct{}

func (d entryDelegate) Height() int                             { return 2 }
func (d entryDelegate) Spacing() int                            { return 0 }
func (d entryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d entryDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(entryItem)
	if !ok {
		return
	}

	var baseStyle lipgloss.Style
	if index == m.Index() {
		baseStyle = selectedItemStyle
	} else {
		baseStyle = itemStyle
	}

	nameStyle := baseStyle
	typeStyle := baseStyle.Copy().Foreground(lipgloss.Color("240")).Italic(true)
	pathStyle := baseStyle.Copy().Foreground(lipgloss.Color("240")).Italic(true)

	// First line: Name
	fmt.Fprintf(w, "%s\n", nameStyle.Render(i.Name))
	
	// Second line: Type and Path
	fmt.Fprintf(w, "%s %s",
		typeStyle.Render(i.Type),
		pathStyle.Render(i.Path))
}

type EntryModel struct {
	list     list.Model
	document string
	viewport viewport.Model
	ready    bool
	width    int
	height   int
	err      error
	cache    *Cache
	slug     string
}

func NewEntryModel(entries []DocumentEntry, cache *Cache, slug string) EntryModel {
	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = entryItem(entry)
	}

	l := list.New(items, entryDelegate{}, 80, 20)
	l.Title = "Choose an entry"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return EntryModel{
		list:     l,
		viewport: viewport.New(80, 20),
		cache:    cache,
		slug:     slug,
	}
}

func (m EntryModel) Init() tea.Cmd {
	return nil
}

func (m EntryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	// case tea.WindowSizeMsg:
	// 	m.list.SetSize(msg.Width, msg.Height-4)
	// 	return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			selected := m.GetSelected()
			htmlPath := filepath.Join(m.cache.GetHTMLDir(m.slug), strings.ReplaceAll(selected.Path, ".", string(os.PathSeparator))) + ".html"
			
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
		}
		m.list.SetWidth(msg.Width)
		return m, nil
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m EntryModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	view := "\n" + m.list.View()
	if m.err != nil {
		view += "\n\nError: " + m.err.Error()
	}
	return view
}

func (m EntryModel) GetSelected() DocumentEntry {
	if i, ok := m.list.SelectedItem().(entryItem); ok {
		return DocumentEntry(i)
	}
	return DocumentEntry{}
}
