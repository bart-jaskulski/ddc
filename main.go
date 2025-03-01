package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbletea/v2"
	"github.com/urfave/cli/v3"
)

// runView starts a TUI to view documentation entries for a given slug
func runView(slug string) error {
	cache := newCache()
	client := newDocs(cache)

	if !client.IsDocSetInstalled(slug) {
		return cli.Exit(fmt.Sprintf("Documentation %s is not installed. Use 'ddc download %s' first", slug, slug), 1)
	}

	docsets, err := client.GetDocumentation(slug)
	if err != nil {
		return err
	}

	model := NewEntryModel(docsets, cache, slug)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err = p.Run()
	return err
}

// runSearch starts a TUI to search with an optional docset filter
func runSearch(query string, docset ...string) error {
	cache := newCache()

	model, err := NewSearchModel(cache, query, docset...)
	if err != nil {
		return err
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// runDownload starts a TUI to list and download documentation sets
func runDownload(docs string) error {
	cache := newCache()
	client := newDocs(cache)

	docsets, err := ListDocumentations()
	if err != nil {
		return err
	}

	if docs != "" {
		// Filter docsets by provided slugs
		for _, doc := range docsets {
			if doc.Slug == docs {
				if !client.IsDocSetInstalled(docs) {
					client.DownloadDocSet(&doc)
				}
			}
		}
		return nil
	}

	model := NewProviderModel(docsets, cache, client)
	p := tea.NewProgram(model)

	_, err = p.Run()
	return err
}

// runList starts a TUI to list downloaded documentation sets
func runList() error {
	cache := newCache()
	client := newDocs(cache)

	model := NewListModel(cache, client)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

var rootCmd = &cli.Command{
	EnableShellCompletion: true,
	Name:                  "ddc",
	Usage:                 "DevDocs CLI browser",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Implement the smart command logic here
		args := cmd.Args().Slice()
		cache := newCache()
		client := newDocs(cache)

		switch cmd.Args().Len() {
		case 0:
			return runList()
		case 1:
			// One argument - could be a documentation set or a search term
			firstArg := cmd.Args().First()

			// Check if it's an installed documentation set
			if client.IsDocSetInstalled(firstArg) {
				// It's a doc set, view it
				return runView(firstArg)
			} else {
				// Not a doc set, treat as search query across all docs
				return runSearch(firstArg)
			}
		default:
			// Multiple arguments - first arg is the doc set, rest is the search query
			docSet := args[0]
			searchQuery := args[1]

			// For multiple search terms, join them
			for i := 2; i < cmd.Args().Len(); i++ {
				searchQuery += " " + args[i]
			}

			// If the doc set exists, search within it
			if client.IsDocSetInstalled(docSet) {
				// Search within the specified doc set
				return runSearch(searchQuery, docSet)
			} else {
				// If doc set doesn't exist, treat all args as a search query
				fullQuery := args[0]
				for i := 1; i < cmd.Args().Len(); i++ {
					fullQuery += " " + args[i]
				}

				return runSearch(fullQuery)
			}
		}
	},
	Commands: []*cli.Command{
		{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "Search across all installed documentation sets",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				if cmd.Args().Len() != 1 {
					return cli.Exit("Please provide a search query", 1)
				}
				query := cmd.Args().First()
				return runSearch(query)
			},
		},
		{
			Name:    "download",
			Aliases: []string{"dl"},
			Usage:   "List and download documentation sets",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runDownload(cmd.Args().First())
			},
		},
		{
			Name:    "view",
			Aliases: []string{"v"},
			Usage:   "View documentation entries",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				if cmd.Args().Len() != 1 {
					return cli.Exit("Please provide a documentation name (e.g., wordpress)", 1)
				}
				slug := cmd.Args().First()
				return runView(slug)
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List downloaded documentation sets",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return runList()
			},
		},
	},
}

func main() {
	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
