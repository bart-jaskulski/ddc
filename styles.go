package main

import (
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = itemStyle.Foreground(lipgloss.Color("10"))
	paginationStyle   = list.DefaultStyles(false).PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles(false).HelpStyle.PaddingLeft(4).PaddingBottom(1)
	focusedStyle      = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39"))
)
