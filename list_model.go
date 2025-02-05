package main

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"fmt"
	"os"
)

type docItem struct {
	slug string
}

func (i docItem) FilterValue() string { return i.slug }

type docListDelegate struct{}

func (d docListDelegate) Height() int                             { return 1 }
func (d docListDelegate) Spacing() int                            { return 0 }
func (d docListDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d docListDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(docItem)
	if !ok {
		return
	}

	str := i.slug

	fn := itemStyle.Render
	if index == m.Index() {
		fn = selectedItemStyle.Render
	}

	fmt.Fprintf(w, fn(str))
}

type ListModel struct {
	list     list.Model
	cache    *Cache
	client   *DevDoc
	quitting bool
  width, height int
}

func NewListModel(cache *Cache, client *DevDoc) ListModel {
	// Read all entries in the cache directory
	entries, err := os.ReadDir(cache.BaseDir)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	items := make([]list.Item, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			items = append(items, docItem{slug: entry.Name()})
		}
	}

	l := list.New(items, docListDelegate{}, 80, 20)
	l.Title = "Downloaded Documentation Sets"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.FilterPrompt = focusedStyle
	l.Styles.FilterCursor = focusedStyle

	return ListModel{
		list:   l,
		cache:  cache,
		client: client,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
  case tea.WindowSizeMsg:
      m.width = msg.Width
      m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(docItem); ok {
				docsets, err := m.client.GetDocumentation(i.slug)
				if err != nil {
					return m, nil
				}
				model := NewEntryModel(docsets, m.cache, i.slug)
				var cmds []tea.Cmd
				cmd := model.Init()
				cmds = append(cmds, cmd)
				
				newModel, cmd := model.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
				if entryModel, ok := newModel.(EntryModel); ok {
					model = entryModel
				}
				cmds = append(cmds, cmd)
				return model, tea.Batch(cmds...)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ListModel) View() string {
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}
