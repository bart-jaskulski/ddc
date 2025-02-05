package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type docItem Documentation

func (i docItem) FilterValue() string { return i.Name }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(docItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s - %s", i.Name, i.Version)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type ChooseModel struct {
	list     list.Model
	selected map[string]bool
}

func NewChooseModel(docsets []Documentation) ChooseModel {
	items := make([]list.Item, len(docsets))
	for i, ds := range docsets {
		items[i] = docItem(ds)
	}

	l := list.New(items, itemDelegate{}, 80, 20)
	l.Title = "Choose docsets to download"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return ChooseModel{
		list:     l,
		selected: make(map[string]bool),
	}
}

func (m ChooseModel) Init() tea.Cmd {
	return nil
}

func (m ChooseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "space":
			if i, ok := m.list.SelectedItem().(docItem); ok {
				m.selected[i.Slug] = !m.selected[i.Slug]
			}
			return m, nil
		case "enter":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ChooseModel) View() string {
	return "\n" + m.list.View()
}

func (m ChooseModel) GetSelected() []Documentation {
	var selected []Documentation
	for i := 0; i < len(m.list.Items()); i++ {
		if item, ok := m.list.Items()[i].(docItem); ok {
			if m.selected[item.Slug] {
				selected = append(selected, Documentation(item))
			}
		}
	}
	return selected
}
