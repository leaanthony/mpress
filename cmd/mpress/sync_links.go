package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/translate"
)

func syncTranslationLinks(args []string) error {
	fs := flag.NewFlagSet("translate sync-links", flag.ContinueOnError)
	language := fs.String("lang", "", "one configured target language")
	file := fs.String("file", "", "English Markdown page relative to content directory")
	previous := fs.String("previous-source", "", "previous English Markdown file to compare with current source")
	write := fs.Bool("write", false, "apply link changes; otherwise preview without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *language == "" || *file == "" || *previous == "" {
		return fmt.Errorf("usage: mpress translate sync-links --lang CODE --file PAGE --previous-source FILE [--write]")
	}
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	baseline, err := os.ReadFile(*previous)
	if err != nil {
		return err
	}
	changed, err := translate.NewEngine(root, cfg, nil).SyncLinks(*language, *file, baseline, *write)
	if err != nil {
		return err
	}
	fmt.Printf("%s/%s: link changes=%t, write=%t; translation state and audit exceptions unchanged\n", *language, *file, changed, *write)
	return nil
}
