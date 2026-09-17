package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/translate"
)

func checkCommand(args []string) error {
	if len(args) == 0 {
		return checkSite()
	}
	if args[0] != "site" {
		return fmt.Errorf("usage: mpress check [site [--output DIRECTORY] [--cloudflare-pages] [--json]]")
	}
	fs := flag.NewFlagSet("check site", flag.ContinueOnError)
	output := fs.String("output", "", "built site directory (relative to project root)")
	cloudflare := fs.Bool("cloudflare-pages", false, "enforce the Pages 20,000 file and 25 MiB per-file limits")
	jsonOut := fs.Bool("json", false, "print JSON result")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("mpress check site accepts flags only")
	}
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if *output != "" && !filepath.IsAbs(*output) {
		*output = filepath.Join(root, *output)
	}
	report, err := check.Rendered(root, cfg, check.RenderedOptions{Output: *output, CloudflarePages: *cloudflare})
	if err != nil {
		return err
	}
	summary := fmt.Sprintf("Checked %d file(s), %d HTML page(s), and %d redirect(s)", report.Files, report.HTMLPages, report.Redirects)
	return printValidationReport(report, report.Errors, summary, *jsonOut)
}

func checkTranslations(args []string) error {
	fs := flag.NewFlagSet("translate check", flag.ContinueOnError)
	language := fs.String("lang", "", "target language (defaults to all configured targets)")
	exceptions := fs.String("exceptions", "", "JSON array of hash-pinned audit exceptions, relative to project root")
	var excluded []string
	fs.Func("exclude-audit", "source file to skip in audits only; repeat for multiple legacy files", func(value string) error { excluded = append(excluded, value); return nil })
	jsonOut := fs.Bool("json", false, "print JSON result")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("mpress translate check accepts flags only")
	}
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	options := translate.CheckOptions{Language: *language, ExcludeAudit: excluded}
	if *exceptions != "" {
		path := *exceptions
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &options.Exceptions); err != nil {
			return fmt.Errorf("read audit exceptions: %w", err)
		}
	}
	report, err := translate.NewEngine(root, cfg, nil).Check(options)
	if err != nil {
		return err
	}
	summary := fmt.Sprintf("Checked %d language(s); accepted %d reviewed finding(s)", len(report.Languages), report.Accepted)
	if len(report.ExcludedAudit) > 0 {
		summary += "; excluded from audit: " + strings.Join(report.ExcludedAudit, ", ")
	}
	return printValidationReport(report, report.Errors, summary, *jsonOut)
}

func printValidationReport(report any, problems []string, summary string, jsonOut bool) error {
	if jsonOut {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
	} else {
		for _, problem := range problems {
			fmt.Println(problem)
		}
		fmt.Printf("%s: %d error(s).\n", summary, len(problems))
	}
	if len(problems) > 0 {
		return fmt.Errorf("validation failed with %d error(s)", len(problems))
	}
	return nil
}
