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
      Name:   "choose",
      Usage: "Choose and download docsets",
      Action: func(ctx context.Context, cmd *cli.Command) error {
        return runChoose(cmd.Args().Slice())
      },
    },
  },
}

func runDoc() error {
  cache := newCache()
  client := newDocs(cache)

  docsets, err := client.GetDocumentation("wordpress")
  if err != nil {
    return err
  }

  model := NewEntryModel(docsets)
  p := tea.NewProgram(model, tea.WithAltScreen())

  _, err = p.Run()
  return err
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
    model := NewChooseModel(docsets)
    p := tea.NewProgram(model)

    finalModel, err := p.Run()
    if err != nil {
      return err
    }

    if m, ok := finalModel.(ChooseModel); ok {
      selected = m.GetSelected()
    }
  }

  // Download selected docsets
  for _, ds := range selected {
    if err := client.DownloadDocSet(ds); err != nil {
      return fmt.Errorf("failed to download %s: %w", ds.Slug, err)
    }
    fmt.Printf("Downloaded %s successfully\n", ds.GetDisplayName())
  }

  return nil
}

func main() {
  if err := rootCmd.Run(context.Background(), os.Args); err != nil {
    log.Fatal(err)
    os.Exit(1)
  }
}
