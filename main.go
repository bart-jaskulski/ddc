package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbletea/v2"
	"github.com/urfave/cli/v3"
)

var rootCmd = &cli.Command{
  EnableShellCompletion: true,
	Name:  "ddc",
	Usage: "DevDocs CLI browser",
	// Action: func(context.Context, *cli.Command) error {
	// 	return runDoc()
	// },
	Commands: []*cli.Command{
		{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "Search across all installed documentation sets",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				if len(cmd.Args().Slice()) != 1 {
					return cli.Exit("Please provide a search query", 1)
				}
				query := cmd.Args().First()
				cache := newCache()

				model, err := NewSearchModel(cache, query)
				if err != nil {
					return err
				}

				p := tea.NewProgram(model, tea.WithAltScreen())
				_, err = p.Run()
				return err
			},
		},
		{
			Name:    "download",
			Aliases: []string{"dl"},
			Usage:   "List and download documentation sets",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cache := newCache()
				client := newDocs(cache)

				docsets, err := ListDocumentations()
				if err != nil {
					return err
				}

				model := NewProviderModel(docsets, cache, client)
				p := tea.NewProgram(model)

				_, err = p.Run()
				if err != nil {
					return err
				}

				return nil
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
				if err != nil {
					return err
				}

				return nil

			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List downloaded documentation sets",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cache := newCache()
				client := newDocs(cache)

				model := NewListModel(cache, client)
				p := tea.NewProgram(model, tea.WithAltScreen())

				_, err := p.Run()
				return err
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
