package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
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
	typeStyle := baseStyle.Italic(true)
	pathStyle := baseStyle.Italic(true)

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

	l := list.New(items, entryDelegate{}, 80, 30)
	l.Title = "Choose an entry"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return EntryModel{
		list:  l,
		cache: cache,
		slug:  slug,
	}
}

func (m EntryModel) Init() (tea.Model, tea.Cmd) {
	return m, nil
}

func (m EntryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "o":
			if m.list.SettingFilter() {
				break
			}
			selected := m.GetSelected()
			
			// Get the file path and fragment
			htmlPath, fragment := m.cache.GetHTMLPath(m.slug, selected.Path)
			
			// Pass the path with fragment to lynx
			// args := []string{htmlPath}
			// if fragment != "" {
			// 	// For lynx, we can use -jumpfile to jump to a specific fragment
			// 	args = append(args, "-jumpfile", fragment[1:]) // Remove the # from fragment
			// }
			
			cmd := exec.Command("lynx", htmlPath + "#" + fragment)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				m.err = fmt.Errorf("failed to open documentation: %w, trying %s", err, htmlPath)
				return m, nil
			}
			return m, tea.Quit
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m EntryModel) View() string {
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
