package main

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/importer"
	"github.com/leaanthony/mpress/internal/translate"
)

func migrateTranslationState(args []string) error {
	flags := flag.NewFlagSet("translate migrate-state", flag.ContinueOnError)
	from := flags.String("from", "", "project snapshot containing the original source, translated files and sidecars")
	language := flags.String("lang", "", "target language (all configured languages by default)")
	file := flags.String("file", "", "current source page relative to the content directory")
	write := flags.Bool("write", false, "write migrated sidecars after validating every selected triplet (default: dry run)")
	jsonOut := flags.Bool("json", false, "print the migration report as JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *from == "" || flags.NArg() != 0 {
		return errors.New("usage: mpress translate migrate-state --from ORIGINAL_PROJECT [--lang CODE] [--file PAGE] [--write] [--json]")
	}
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	original, err := filepath.Abs(*from)
	if err != nil {
		return err
	}
	engine := translate.NewEngine(root, cfg, nil)
	engine.MarkdownToMPD = importer.MarkdownToMPD
	report, err := engine.MigrateState(original, *language, *file, *write)
	if err != nil {
		return err
	}
	if *jsonOut {
		fmt.Println(report.JSON())
		return nil
	}
	for _, item := range report.Files {
		if item.Unchanged {
			fmt.Printf("%s/%s: already migrated\n", item.Language, item.File)
		} else {
			fmt.Printf("%s/%s: %d segments, %d require review\n", item.Language, item.File, item.Segments, item.Conflicts)
		}
	}
	if !*write {
		fmt.Println("Dry run. Add --write to save sidecars; source and translated content remain untouched.")
	} else {
		fmt.Printf("Wrote %d translation sidecars.\n", report.Written)
	}
	return nil
}
