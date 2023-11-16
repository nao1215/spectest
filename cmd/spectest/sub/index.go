package sub

import (
	"fmt"
	"path/filepath"

	"github.com/go-spectest/markdown"
	"github.com/spf13/cobra"
)

// newIndexCmd return index command.
func newIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "index",
		Short:   "Generate an index for a directory full of markdown files",
		RunE:    index,
		Example: "   spectest index TARGET_DIR",
	}

	cmd.Flags().StringP("title", "t", "", "title of index markdown file")
	cmd.Flags().StringSliceP("desc", "d", []string{}, "description of index markdown file")
	return cmd
}

// indexer is a struct for index command.
// It has a target directory and a title of index.
type indexer struct {
	// target is a target directory.
	target string
	// title is a title of index.
	title string
	// description is a description of index.
	description []string
}

// newIndexer return indexer.
func newIndexer(cmd *cobra.Command, args []string) (*indexer, error) {
	title, err := cmd.Flags().GetString("title")
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetStringSlice("desc")
	if err != nil {
		return nil, err
	}

	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	return &indexer{
		target:      target,
		title:       title,
		description: description,
	}, nil
}

// run generate an index for a directory full of markdown files.
func (i *indexer) run() error {
	if err := markdown.GenerateIndex(i.target, markdown.WithTitle(i.title), markdown.WithDescription(i.description)); err != nil {
		return err
	}
	fmt.Printf("generated index markdown at %s\n", filepath.Join(i.target, "index.md"))
	return nil
}

// index generate an index for a directory full of markdown files.
func index(cmd *cobra.Command, args []string) error {
	i, err := newIndexer(cmd, args)
	if err != nil {
		return fmt.Errorf("failed to initialize index command: %w", err)
	}
	return i.run()
}
