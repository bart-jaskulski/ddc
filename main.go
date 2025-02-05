package main

import (
  "fmt"
  "os"
  "context"
  "log"

  "github.com/charmbracelet/bubbletea"
  "github.com/urfave/cli/v3"
)

var rootCmd = &cli.Command{
  Name:   "dd",
  Usage: "DevDocs CLI browser",
  Action: func(context.Context, *cli.Command) error {
    return runDoc()
  },
  Commands: []*cli.Command{
    {
      Name:   "download",
      Aliases: []string{"dl"},
      Usage:  "List and download documentation sets",
      Action: func(ctx context.Context, cmd *cli.Command) error {
        return runChoose(cmd.Args().Slice())
      },
    },
    {
      Name:   "view",
      Aliases: []string{"v"},
      Usage:  "View documentation entries",
      Action: func(ctx context.Context, cmd *cli.Command) error {
        if len(cmd.Args().Slice()) != 1 {
          return cli.Exit("Please provide a documentation name (e.g., wordpress)", 1)
        }
        return runView(cmd.Args().First())
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

func runView(slug string) error {
  cache := newCache()
  client := newDocs(cache)

  if !client.IsDocSetInstalled(slug) {
    return cli.Exit(fmt.Sprintf("Documentation %s is not installed. Use 'dd download %s' first", slug, slug), 1)
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
}

func runList() error {
	cache := newCache()
	client := newDocs(cache)

	model := NewListModel(cache, client)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

func runDoc() error {
  return nil
  // cache := newCache()
  // client := newDocs(cache)
  //
  // docsets, err := client.GetDocumentation("wordpress")
  // if err != nil {
  //   return err
  // }
  //
  // model := NewEntryModel(docsets)
  // p := tea.NewProgram(model, tea.WithAltScreen())
  //
  // _, err = p.Run()
  // return err
}

func runChoose(args []string) error {
  cache := newCache()
  client := newDocs(cache)

  docsets, err := ListDocumentations()
  if err != nil {
    return err
  }

  var selected []Documentation
  if len(args) > 0 {
    // Filter docsets based on provided arguments
    for _, arg := range args {
      for _, ds := range docsets {
        if ds.Slug == arg {
          selected = append(selected, ds)
          break
        }
      }
    }
  } else {
    // If no args, show interactive selection
    model := NewProviderModel(docsets, cache, client)
    p := tea.NewProgram(model)

    finalModel, err := p.Run()
    if err != nil {
      return err
    }

    if m, ok := finalModel.(ProviderModel); ok {
      selected = m.GetSelected()
    }
  }


  return nil
}

func main() {
  if err := rootCmd.Run(context.Background(), os.Args); err != nil {
    log.Fatal(err)
    os.Exit(1)
  }
}
