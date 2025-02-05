package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type entryItem DocEntry

func (i entryItem) FilterValue() string { return i.Name }

type entryDelegate struct{}

func (d entryDelegate) Height() int                             { return 1 }
func (d entryDelegate) Spacing() int                            { return 0 }
func (d entryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d entryDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(entryItem)
	if !ok {
		return
	}

	str := i.Name

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type EntryModel struct {
	list     list.Model
	document string
	viewport viewport.Model
	ready    bool
	width    int
	height   int
}

func NewEntryModel(entries []DocEntry) EntryModel {
	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = entryItem(entry)
	}

	l := list.New(items, entryDelegate{}, 80, 20)
	l.Title = "Choose an entry"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return EntryModel{
		list:     l,
		viewport: viewport.New(80, 20),
	}
}

func (m EntryModel) Init() tea.Cmd {
	return nil
}

func (m EntryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
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

	return "\n" + m.list.View()
}

func (m EntryModel) GetSelected() DocEntry {
	if i, ok := m.list.SelectedItem().(entryItem); ok {
		return DocEntry(i)
	}
	return DocEntry{}
}
