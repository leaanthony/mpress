package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/contribute"
	"github.com/leaanthony/mpress/internal/dev"
	"github.com/leaanthony/mpress/internal/exportzip"
	"github.com/leaanthony/mpress/internal/importer"
	"github.com/leaanthony/mpress/internal/knowledge"
	"github.com/leaanthony/mpress/internal/operations"
	"github.com/leaanthony/mpress/internal/projectconvert"
	"github.com/leaanthony/mpress/internal/quickedit"
	"github.com/leaanthony/mpress/internal/site"
	"github.com/leaanthony/mpress/internal/translate"
	docversion "github.com/leaanthony/mpress/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "init":
		return initProject(args[1:])
	case "build":
		return build(args[1:])
	case "export":
		return exportSite(args[1:])
	case "dev":
		return serve(args[1:])
	case "contribute":
		return contributeSite(args[1:])
	case "clean":
		return clean()
	case "check":
		return checkCommand(args[1:])
	case "import":
		return importSite(args[1:])
	case "convert":
		return convertSite(args[1:])
	case "deploy":
		return deploySite(args[1:])
	case "versions":
		return versions(args[1:])
	case "translate", "translations":
		return translateSite(args[1:])
	case "knowledge":
		return serveKnowledge(args[1:])
	case "version", "--version", "-v":
		fmt.Println("mpress", cliVersion())
		return nil
	case "help", "--help", "-h":
		return usage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
func usage() error {
	fmt.Print(`M-Press: a small Markdown documentation generator

Usage:
  mpress init [directory]
  mpress build [--strict] [--drafts] [--json] [--no-purge-css]
  mpress export [--output archive.zip] [--strict] [--drafts] [--force] [--json] [archive.zip]
  mpress dev [directory] [--host 127.0.0.1] [--port 3000] [--read-only]
  mpress dev --repo URL --checkout DIRECTORY [--branch docs/mpress]
  mpress contribute <site-url> [--goal page|translate] [--checkout DIRECTORY] [--port 3000] [--draft-file FILE] [--no-open]
  mpress clean
  mpress check
  mpress check site [--output DIRECTORY] [--cloudflare-pages] [--json]
  mpress import --from starlight <source> --output <directory>
	  mpress convert [--to mpd|markdown] [--content DIRECTORY] --replace
  mpress deploy [--target name] [--production] [--json]
  mpress deploy configure cloudflare --account ID --project NAME [--target name]
  mpress deploy configure netlify --site ID_OR_DOMAIN [--account SLUG] [--target name]
  mpress versions capture|list|verify|remove
  mpress translate [status] [--lang CODE] [--add-language] [--label NAME] [--file PAGE] [--scope missing|stale|all] [--harness codex|claudecode] [--model MODEL] [--estimate] [--workers N]
  mpress translate audit --lang CODE [--file PAGE] [--json]
  mpress translate check [--lang CODE] [--exceptions FILE] [--exclude-audit PAGE] [--json]
  mpress translate review --lang CODE --file PAGE [--status reviewed|final]
  mpress translate --repo URL --checkout DIRECTORY [--branch NAME] [translation options]
  mpress knowledge [directory] [--transport stdio|http] [--host 127.0.0.1] [--port 3100]
  mpress knowledge evaluate [directory] --suite FILE [--json]
  mpress version
`)
	return nil
}

func translateSite(args []string) error {
	if len(args) > 0 && args[0] == "check" {
		return checkTranslations(args[1:])
	}
	statusOnly := len(args) > 0 && args[0] == "status"
	reviewOnly := len(args) > 0 && args[0] == "review"
	auditOnly := len(args) > 0 && args[0] == "audit"
	if statusOnly || reviewOnly || auditOnly {
		args = args[1:]
	}
	fs := flag.NewFlagSet("translate", flag.ContinueOnError)
	language := fs.String("lang", "", "target language code")
	addLanguage := fs.Bool("add-language", false, "add --lang to site.languages before translating")
	languageLabel := fs.String("label", "", "reader-facing label for --add-language (defaults to the language code)")
	file := fs.String("file", "", "source page relative to the content directory")
	scope := fs.String("scope", "stale", "translation scope: missing, stale, or all")
	dryRun := fs.Bool("dry-run", false, "report work without contacting the provider or writing files")
	estimate := fs.Bool("estimate", false, "estimate requests, tokens, and configured cost without contacting the provider or writing files")
	harness := fs.String("harness", "", "use a local authenticated harness for this run: codex or claudecode")
	model := fs.String("model", "", "override the translation model for this run")
	force := fs.Bool("force", false, "replace conflicting manual translations")
	workers := fs.Int("workers", 4, "number of pages to translate concurrently")
	jsonOut := fs.Bool("json", false, "print JSON result")
	reviewStatus := fs.String("status", "reviewed", "review status: reviewed or final")
	repository := fs.String("repo", "", "Git repository to clone before translating")
	checkout := fs.String("checkout", "", "local checkout directory for --repo")
	branch := fs.String("branch", "", "Git branch for --repo")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: mpress translate [status] [--lang CODE] [--add-language] [--label NAME] [--file PAGE] [--scope missing|stale|all] [--harness codex|claudecode] [--model MODEL] [--estimate] [--workers N]")
	}
	if *workers < 1 || *workers > 16 {
		return errors.New("--workers must be between 1 and 16")
	}
	var root string
	var err error
	if strings.TrimSpace(*repository) != "" {
		if strings.TrimSpace(*checkout) == "" {
			return errors.New("--checkout is required with --repo")
		}
		root, err = translate.CloneProject(context.Background(), *repository, *branch, *checkout)
		if err == nil && !*jsonOut {
			fmt.Println("Cloned translation checkout to", root)
		}
	} else {
		root, err = project()
	}
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if statusOnly || reviewOnly {
		if strings.TrimSpace(*harness) != "" || strings.TrimSpace(*model) != "" {
			return errors.New("--harness and --model are only valid when planning or running a translation")
		}
	} else if err := applyTranslationHarness(&cfg, *harness); err != nil {
		return err
	}
	if selectedModel := strings.TrimSpace(*model); selectedModel != "" {
		cfg.Translation.Model = selectedModel
	}
	if *addLanguage {
		if reviewOnly || statusOnly || auditOnly {
			return errors.New("--add-language is only valid when translating")
		}
		if strings.TrimSpace(*language) == "" {
			return errors.New("--add-language requires --lang CODE")
		}
		canonicalLanguage, changed, addErr := addTranslationLanguage(root, &cfg, *language, *languageLabel, !*dryRun && !*estimate)
		if addErr != nil {
			return addErr
		}
		*language = canonicalLanguage
		if changed && !*jsonOut {
			verb := "Added"
			if *dryRun || *estimate {
				verb = "Would add"
			}
			fmt.Printf("%s %s (%s) to site.languages.\n", verb, canonicalLanguage, cfg.Site.LanguageLabels[canonicalLanguage])
		}
	}
	if !reviewOnly && !auditOnly && strings.TrimSpace(*harness) == "" && strings.TrimSpace(cfg.Translation.Command) == "" && (strings.TrimSpace(cfg.Translation.Model) == "" || strings.EqualFold(strings.TrimSpace(cfg.Translation.Model), "auto")) {
		selection, ok := translate.SelectBestTranslationModel(translate.ModelSelectionRequest{
			Task: translate.TaskDraft, TargetLanguage: *language, Capabilities: translate.DetectModelCapabilities(cfg),
		})
		if !ok {
			return errors.New("no authenticated translation option is available; install Codex or Claude Code, or configure an OpenRouter or OpenAI API key")
		}
		translate.ApplyModelSelection(&cfg, selection)
		if !*jsonOut {
			fmt.Printf("Selected %s via %s: %s.\n", selection.Model, selection.Provider, selection.Basis)
		}
	}
	if reviewOnly {
		if *language == "" || *file == "" {
			return errors.New("mpress translate review requires --lang and --file")
		}
		report, markErr := translate.NewEngine(root, cfg, nil).Mark(*language, *file, *reviewStatus)
		if markErr != nil {
			return markErr
		}
		if *jsonOut {
			fmt.Println(translate.MarshalReport(translate.Report{Files: []translate.FileReport{report}, Segments: report.Segments}))
		} else {
			fmt.Printf("Marked %s → %s as %s.\n", report.SourceFile, report.TargetLanguage, *reviewStatus)
		}
		return nil
	}
	if auditOnly {
		if strings.TrimSpace(*language) == "" {
			return errors.New("mpress translate audit requires --lang")
		}
		engine := translate.NewEngine(root, cfg, nil)
		var report translate.AuditReport
		var auditErr error
		if strings.TrimSpace(*harness) != "" {
			reviewer, reviewerErr := translate.NewLocalProvider(cfg.Translation.Provider, cfg.Translation.Model, cfg.Translation.Command)
			if reviewerErr != nil {
				return reviewerErr
			}
			report, auditErr = engine.AuditWithReviewer(context.Background(), *language, *file, reviewer)
		} else {
			if strings.TrimSpace(*model) != "" {
				return errors.New("mpress translate audit --model requires --harness")
			}
			report, auditErr = engine.Audit(*language, *file)
		}
		if auditErr != nil {
			return auditErr
		}
		if *jsonOut {
			fmt.Println(report.JSON())
		} else {
			for _, finding := range report.Findings {
				location := finding.File
				if finding.Segment != "" {
					location += ":" + finding.Segment
				}
				fmt.Printf("%s %s [%s] %s\n", strings.ToUpper(finding.Severity), location, finding.Code, finding.Message)
			}
			fmt.Printf("Audited %d file(s) and %d segment(s): %d error(s), %d warning(s).\n", report.Files, report.Segments, report.Errors, report.Warnings)
		}
		if report.Errors > 0 {
			return fmt.Errorf("translation audit failed with %d error(s)", report.Errors)
		}
		return nil
	}
	options := translate.Options{Language: *language, File: *file, Scope: *scope, DryRun: *dryRun || *estimate || statusOnly, Force: *force, Workers: *workers}
	var engine *translate.Engine
	if options.DryRun {
		engine = translate.NewEngine(root, cfg, nil)
	} else {
		engine, err = translate.NewConfiguredEngine(root, cfg)
		if err != nil {
			return err
		}
	}
	report, err := engine.Run(context.Background(), options)
	if err != nil {
		return err
	}
	if *jsonOut {
		fmt.Println(translate.MarshalReport(report))
		return nil
	}
	for _, fileReport := range report.Files {
		if fileReport.NeedsTranslation == 0 {
			fmt.Printf("%s → %s: up to date", fileReport.SourceFile, fileReport.TargetLanguage)
		} else if options.DryRun {
			fmt.Printf("%s → %s: %d segment(s) need translation", fileReport.SourceFile, fileReport.TargetLanguage, fileReport.NeedsTranslation)
		} else {
			fmt.Printf("%s → %s: translated %d segment(s)", fileReport.SourceFile, fileReport.TargetLanguage, fileReport.Translated)
		}
		if conflicts := fileReport.States["conflict"]; conflicts > 0 {
			fmt.Printf("; %d manual conflict(s) need review or --force", conflicts)
		}
		fmt.Println()
	}
	if options.DryRun {
		fmt.Printf("%d segment(s) need translation across %d page target(s)\n", report.Pending, len(report.Files))
		if *estimate {
			fmt.Printf("Estimated provider work: %d request(s), approximately %d input token(s) and %d output token(s)\n", report.Estimate.Requests, report.Estimate.InputTokens, report.Estimate.OutputTokens)
			if report.Estimate.PricingConfigured {
				fmt.Printf("Estimated cost: $%.4f USD ($%.4f input + $%.4f output)\n", report.Estimate.TotalCostUSD, report.Estimate.InputCostUSD, report.Estimate.OutputCostUSD)
			} else {
				fmt.Println("Estimated cost: unavailable. Set translation.inputPricePerMillion and translation.outputPricePerMillion for the configured model.")
			}
			fmt.Println("Estimate only. No provider was contacted and no files were written.")
		}
	} else {
		fmt.Printf("Wrote %d translated page(s).\n", report.Written)
	}
	return nil
}

func applyTranslationHarness(cfg *config.Config, requested string) error {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "" {
		return nil
	}
	switch requested {
	case "codex":
		cfg.Translation.Provider = "codex"
		cfg.Translation.Command = "codex"
		cfg.Translation.Model = "gpt-5.6-sol"
	case "claudecode", "claude-code", "claude":
		cfg.Translation.Provider = "claude"
		cfg.Translation.Command = "claude"
		cfg.Translation.Model = "claude-opus-4.6"
	default:
		return fmt.Errorf("unknown translation harness %q; use codex or claudecode", requested)
	}
	return nil
}

func addTranslationLanguage(root string, cfg *config.Config, language, label string, persist bool) (string, bool, error) {
	language, err := canonicalLanguageCode(language)
	if err != nil {
		return "", false, err
	}
	if strings.EqualFold(language, cfg.Site.DefaultLanguage) {
		return "", false, errors.New("target language must differ from the default language")
	}
	for _, existing := range cfg.Site.Languages {
		if strings.EqualFold(existing, language) {
			changed := false
			label = strings.TrimSpace(label)
			if label != "" && cfg.Site.LanguageLabels[existing] != label {
				if cfg.Site.LanguageLabels == nil {
					cfg.Site.LanguageLabels = map[string]string{}
				}
				cfg.Site.LanguageLabels[existing] = label
				changed = true
			}
			if changed && persist {
				if err := config.Save(root, *cfg); err != nil {
					return "", false, fmt.Errorf("update translation language: %w", err)
				}
			}
			return existing, changed, nil
		}
	}
	cfg.Site.Languages = append(cfg.Site.Languages, language)
	if cfg.Site.LanguageLabels == nil {
		cfg.Site.LanguageLabels = map[string]string{}
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = language
	}
	cfg.Site.LanguageLabels[language] = label
	if persist {
		if err := config.Save(root, *cfg); err != nil {
			return "", false, fmt.Errorf("add translation language: %w", err)
		}
	}
	return language, true, nil
}

func canonicalLanguageCode(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") || strings.Contains(value, "--") {
		return "", errors.New("language codes use letters, numbers and single internal hyphens")
	}
	parts := strings.Split(value, "-")
	for index, part := range parts {
		for _, character := range part {
			if (character < '0' || character > '9') && (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') {
				return "", errors.New("language codes use letters, numbers and single internal hyphens")
			}
		}
		lower := strings.ToLower(part)
		switch {
		case index == 0:
			parts[index] = lower
		case len(part) == 4 && part[0] >= 'A' && part[0] <= 'z':
			parts[index] = strings.ToUpper(lower[:1]) + lower[1:]
		case len(part) == 2:
			parts[index] = strings.ToUpper(part)
		default:
			parts[index] = lower
		}
	}
	return strings.Join(parts, "-"), nil
}

func deploySite(args []string) error {
	if len(args) > 0 && args[0] == "configure" {
		return configureDeploy(args[1:])
	}
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	target := fs.String("target", "", "deployment target")
	production := fs.Bool("production", false, "deploy to production")
	jsonOut := fs.Bool("json", false, "print JSON result")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := project()
	if err != nil {
		return err
	}
	environment := "preview"
	if *production {
		environment = "production"
	}
	result, err := operations.Deploy(context.Background(), root, *target, environment)
	if err != nil {
		return err
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Deployed %s to %s\n%s\n", result.Target, result.Environment, result.URL)
	}
	return nil
}

func configureDeploy(args []string) error {
	if len(args) == 0 || (args[0] != "cloudflare" && args[0] != "netlify") {
		return errors.New("usage: mpress deploy configure cloudflare --account ID --project NAME [--target name] | mpress deploy configure netlify --site ID_OR_DOMAIN [--account SLUG] [--target name]")
	}
	provider := args[0]
	fs := flag.NewFlagSet("deploy configure "+provider, flag.ContinueOnError)
	account := fs.String("account", "", "Cloudflare account ID")
	projectName := fs.String("project", "", "Cloudflare Pages project or Netlify site name")
	siteID := fs.String("site", "", "Netlify site ID, domain, or name")
	targetName := fs.String("target", provider, "deployment target name")
	branch := fs.String("production-branch", "main", "production branch")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if provider == "cloudflare" && (strings.TrimSpace(*account) == "" || strings.TrimSpace(*projectName) == "") {
		return errors.New("--account and --project are required for Cloudflare")
	}
	if provider == "netlify" && strings.TrimSpace(*siteID) == "" && strings.TrimSpace(*projectName) == "" {
		return errors.New("--site or --project is required for Netlify")
	}
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if cfg.Deploy.Targets == nil {
		cfg.Deploy.Targets = map[string]config.DeployTarget{}
	}
	cfg.Deploy.Default = *targetName
	if provider == "cloudflare" {
		cfg.Deploy.Targets[*targetName] = config.DeployTarget{
			Provider: "cloudflare-pages", AccountID: strings.TrimSpace(*account), Project: strings.TrimSpace(*projectName), ProductionBranch: strings.TrimSpace(*branch),
		}
	} else {
		name := strings.TrimSpace(*siteID)
		if name == "" {
			name = strings.TrimSpace(*projectName)
		}
		cfg.Deploy.Targets[*targetName] = config.DeployTarget{Provider: "netlify", AccountID: strings.TrimSpace(*account), Project: name, ProductionBranch: strings.TrimSpace(*branch)}
	}
	if err := config.Save(root, cfg); err != nil {
		return err
	}
	if provider == "cloudflare" {
		fmt.Printf("Configured Cloudflare target %s. Set CLOUDFLARE_API_TOKEN before deploying.\n", *targetName)
	} else {
		fmt.Printf("Configured Netlify target %s. Set NETLIFY_AUTH_TOKEN before deploying.\n", *targetName)
	}
	return nil
}
func project() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return config.FindProject(cwd)
}
func build(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	strict := fs.Bool("strict", false, "fail on unsupported content")
	drafts := fs.Bool("drafts", false, "include drafts")
	purgeCSS := fs.Bool("purge-css", true, "remove clearly unused CSS selectors from production assets")
	noPurgeCSS := fs.Bool("no-purge-css", false, "retain all CSS selectors in production assets")
	jsonOut := fs.Bool("json", false, "print JSON result")
	cpuProfile := fs.String("cpuprofile", "", "write a CPU profile to this file")
	heapProfile := fs.String("memprofile", "", "write a heap profile to this file after the build")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := project()
	if err != nil {
		return err
	}
	var cpuFile *os.File
	if strings.TrimSpace(*cpuProfile) != "" {
		cpuFile, err = os.Create(*cpuProfile)
		if err != nil {
			return fmt.Errorf("create CPU profile: %w", err)
		}
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			return fmt.Errorf("start CPU profile: %w", err)
		}
	}
	result, buildErr := site.Build(root, site.BuildOptions{Strict: *strict, IncludeDrafts: *drafts, MinifyAssets: true, PurgeUnusedCSS: *purgeCSS && !*noPurgeCSS})
	if cpuFile != nil {
		pprof.StopCPUProfile()
		if err := cpuFile.Close(); err != nil && buildErr == nil {
			buildErr = fmt.Errorf("close CPU profile: %w", err)
		}
	}
	if strings.TrimSpace(*heapProfile) != "" {
		runtime.GC()
		file, profileErr := os.Create(*heapProfile)
		if profileErr == nil {
			profileErr = pprof.WriteHeapProfile(file)
			if closeErr := file.Close(); profileErr == nil {
				profileErr = closeErr
			}
		}
		if profileErr != nil && buildErr == nil {
			buildErr = fmt.Errorf("write heap profile: %w", profileErr)
		}
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		for _, d := range result.Diagnostics {
			fmt.Fprintf(os.Stderr, "%s: %s: %s\n", strings.ToUpper(d.Severity), d.File, d.Message)
		}
		fmt.Printf("Built %d pages and %d files in %s\n", result.Pages, result.Files, result.Duration.Round(1e6))
	}
	return buildErr
}

func exportSite(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	output := fs.String("output", "", "ZIP archive path")
	strict := fs.Bool("strict", false, "fail on unsupported content")
	drafts := fs.Bool("drafts", false, "include drafts")
	force := fs.Bool("force", false, "replace an existing archive")
	jsonOut := fs.Bool("json", false, "print JSON result")
	if err := fs.Parse(interspersedExportArgs(args)); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return errors.New("usage: mpress export [--output archive.zip] [--strict] [--drafts] [--force] [--json] [archive.zip]")
	}
	if fs.NArg() == 1 {
		if strings.TrimSpace(*output) != "" {
			return errors.New("use either the archive argument or --output, not both")
		}
		*output = fs.Arg(0)
	}
	root, err := project()
	if err != nil {
		return err
	}
	if strings.TrimSpace(*output) == "" {
		*output = filepath.Join(root, exportzip.DefaultFilename(root))
	}
	result, err := exportzip.Create(root, *output, exportzip.Options{Strict: *strict, IncludeDrafts: *drafts, Overwrite: *force})
	if err != nil {
		return err
	}
	if *jsonOut {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Exported %d pages and %d files to %s (%d bytes)\n", result.Pages, result.Files, result.Archive, result.Bytes)
	return nil
}

// interspersedExportArgs keeps the explicit archive path convenient without
// forcing users to remember that the standard flag package stops parsing at
// the first positional argument. The output flag is the only export option
// that consumes a following value; all other flags are boolean or use =value.
func interspersedExportArgs(args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			positionals = append(positionals, args[index+1:]...)
			break
		}
		if !strings.HasPrefix(argument, "-") || argument == "-" {
			positionals = append(positionals, argument)
			continue
		}
		flags = append(flags, argument)
		if argument == "--output" || argument == "-output" {
			if index+1 < len(args) {
				index++
				flags = append(flags, args[index])
			}
		}
	}
	return append(flags, positionals...)
}

func serve(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	port := fs.Int("port", 3000, "port")
	host := fs.String("host", "127.0.0.1", "host to bind")
	authoring := fs.Bool("authoring", true, "enable local authoring API writes")
	readOnly := fs.Bool("read-only", false, "disable configuration and source writes")
	token := fs.String("token", "", "token for remote authoring API writes")
	repository := fs.String("repo", "", "Git repository to clone before starting development")
	checkout := fs.String("checkout", "", "local checkout directory for --repo")
	branch := fs.String("branch", "docs/mpress", "safe work branch to create for --repo")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return errors.New("usage: mpress dev [directory] or mpress dev --repo URL --checkout DIRECTORY [--branch NAME]")
	}
	root, err := prepareDevProject(strings.TrimSpace(*repository), strings.TrimSpace(*checkout), strings.TrimSpace(*branch), fs.Args())
	if err != nil {
		return err
	}
	return dev.Serve(root, dev.Options{Host: *host, Port: *port, Authoring: *authoring && !*readOnly, Token: *token, Version: version})
}

func serveKnowledge(args []string) error {
	if len(args) > 0 && (args[0] == "evaluate" || args[0] == "eval") {
		return evaluateKnowledge(args[1:])
	}
	fs := flag.NewFlagSet("knowledge", flag.ContinueOnError)
	transport := fs.String("transport", "stdio", "MCP transport: stdio or http")
	host := fs.String("host", "127.0.0.1", "HTTP host to bind")
	port := fs.Int("port", 3100, "HTTP port")
	token := fs.String("token", "", "optional HTTP bearer token; required outside loopback")
	noBuild := fs.Bool("no-build", false, "serve the existing verified knowledge artifact without rebuilding")
	if err := fs.Parse(interspersedKnowledgeArgs(args)); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return errors.New("usage: mpress knowledge [directory] [--transport stdio|http] [--host 127.0.0.1] [--port 3100] [--token VALUE] [--no-build]")
	}
	root, err := project()
	if fs.NArg() == 1 {
		root, err = filepath.Abs(fs.Arg(0))
		if err == nil {
			root, err = config.FindProject(root)
		}
	}
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if !cfg.Knowledge.Enabled {
		return errors.New("knowledge generation is disabled in mpress.yaml")
	}
	if !*noBuild {
		if _, err := site.Build(root, site.BuildOptions{MinifyAssets: true, PurgeUnusedCSS: true}); err != nil {
			return fmt.Errorf("build knowledge site: %w", err)
		}
	}
	store, err := knowledge.LoadAll(cfg.OutputPath(root))
	if err != nil {
		return fmt.Errorf("load knowledge artifact: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(*transport)) {
	case "stdio":
		return knowledge.RunStdio(context.Background(), store, version)
	case "http":
		if !loopbackHost(*host) && strings.TrimSpace(*token) == "" {
			return errors.New("--token is required when the knowledge server binds outside loopback")
		}
		handler := knowledge.HTTPHandler(store, version)
		if strings.TrimSpace(*token) != "" {
			handler = requireKnowledgeToken(handler, strings.TrimSpace(*token))
		}
		address := net.JoinHostPort(*host, strconv.Itoa(*port))
		mux := http.NewServeMux()
		mux.Handle("/mcp", handler)
		fmt.Printf("M-Press knowledge MCP: http://%s/mcp\n", address)
		return http.ListenAndServe(address, mux)
	default:
		return fmt.Errorf("knowledge transport must be stdio or http, got %q", *transport)
	}
}

func evaluateKnowledge(args []string) error {
	fs := flag.NewFlagSet("knowledge evaluate", flag.ContinueOnError)
	suitePath := fs.String("suite", "", "JSON evaluation suite")
	limit := fs.Int("limit", 5, "maximum search results per case")
	jsonOutput := fs.Bool("json", false, "print the complete JSON report")
	noBuild := fs.Bool("no-build", false, "evaluate the existing verified artifact")
	if err := fs.Parse(interspersedKnowledgeEvaluationArgs(args)); err != nil {
		return err
	}
	if fs.NArg() > 1 || strings.TrimSpace(*suitePath) == "" {
		return errors.New("usage: mpress knowledge evaluate [directory] --suite FILE [--limit 5] [--json] [--no-build]")
	}
	root, err := project()
	if fs.NArg() == 1 {
		root, err = filepath.Abs(fs.Arg(0))
		if err == nil {
			root, err = config.FindProject(root)
		}
	}
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if !cfg.Knowledge.Enabled {
		return errors.New("knowledge generation is disabled in mpress.yaml")
	}
	if !*noBuild {
		if _, err := site.Build(root, site.BuildOptions{MinifyAssets: true, PurgeUnusedCSS: true}); err != nil {
			return fmt.Errorf("build knowledge site: %w", err)
		}
	}
	store, err := knowledge.LoadAll(cfg.OutputPath(root))
	if err != nil {
		return fmt.Errorf("load knowledge artifact: %w", err)
	}
	suite, err := knowledge.LoadEvaluationSuite(*suitePath)
	if err != nil {
		return fmt.Errorf("load knowledge evaluation suite: %w", err)
	}
	report := store.Evaluate(suite, *limit)
	if *jsonOutput {
		data, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("%s: %d/%d passed\n", report.Suite, report.Passed, report.Cases)
	fmt.Printf("Recall@1 %.1f%% · Recall@3 %.1f%% · Recall@5 %.1f%% · MRR %.3f · citations %.1f%%\n", report.RecallAt1*100, report.RecallAt3*100, report.RecallAt5*100, report.MRR, report.CitationRate*100)
	for _, result := range report.Results {
		if result.Passed {
			continue
		}
		top := "no results"
		if len(result.TopURLs) > 0 {
			top = result.TopURLs[0]
		}
		fmt.Printf("FAIL %s: %q; top result %s\n", result.ID, result.Query, top)
	}
	return nil
}

func interspersedKnowledgeArgs(args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			positionals = append(positionals, args[index+1:]...)
			break
		}
		if !strings.HasPrefix(argument, "-") || argument == "-" {
			positionals = append(positionals, argument)
			continue
		}
		flags = append(flags, argument)
		switch argument {
		case "--transport", "-transport", "--host", "-host", "--port", "-port", "--token", "-token":
			if index+1 < len(args) {
				index++
				flags = append(flags, args[index])
			}
		}
	}
	return append(flags, positionals...)
}

func interspersedKnowledgeEvaluationArgs(args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			positionals = append(positionals, args[index+1:]...)
			break
		}
		if !strings.HasPrefix(argument, "-") || argument == "-" {
			positionals = append(positionals, argument)
			continue
		}
		flags = append(flags, argument)
		switch argument {
		case "--suite", "-suite", "--limit", "-limit":
			if index+1 < len(args) {
				index++
				flags = append(flags, args[index])
			}
		}
	}
	return append(flags, positionals...)
}

func loopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func requireKnowledgeToken(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		authorization := strings.TrimSpace(request.Header.Get("Authorization"))
		if len(authorization) < 8 || !strings.EqualFold(authorization[:7], "Bearer ") || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(authorization[7:])), []byte(token)) != 1 {
			response.Header().Set("WWW-Authenticate", `Bearer realm="mpress-knowledge"`)
			http.Error(response, "a valid knowledge server token is required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func contributeSite(args []string) error {
	options, err := parseContributionArgs(args)
	if err != nil {
		return err
	}
	ctx := context.Background()
	metadata, err := contribute.Resolve(ctx, nil, options.SiteURL, options.Branch)
	if err != nil {
		return err
	}
	root, branch, reused, err := contribute.Prepare(ctx, metadata, options.Checkout, time.Now())
	if err != nil {
		return err
	}
	if reused {
		fmt.Println("Reusing contribution checkout", root)
	} else {
		fmt.Println("Created contribution checkout", root)
	}
	fmt.Println("Prepared work branch", branch)
	metadata.Guide = contribute.DetectGuide(root, metadata.Guide)
	if options.DraftFile != "" {
		draftPath, resolveErr := quickedit.ResolveDraftFile(options.DraftFile)
		if resolveErr != nil {
			return fmt.Errorf("find browser draft: %w", resolveErr)
		}
		changes, applyErr := quickedit.ApplyFile(root, metadata.Source, draftPath)
		if applyErr != nil {
			return fmt.Errorf("apply browser draft: %w", applyErr)
		}
		fmt.Printf("Applied %d browser draft change(s) from %s to %s\n", changes, filepath.Base(draftPath), metadata.Source)
	}
	if options.NoOpen {
		fmt.Println("Open the local URL below to use the contribution wizard.")
	} else {
		fmt.Println("The contribution wizard will open in your browser.")
	}
	startCommit := contributionStartCommit(root, metadata.Branch)
	return dev.Serve(root, dev.Options{
		Host: options.Host, Port: options.Port, AutoPort: options.Port == 0, OpenBrowser: !options.NoOpen,
		Authoring: true, Version: version,
		Contribution: &dev.ContributionSession{SiteURL: metadata.SiteURL, Repository: metadata.Repository, SourcePath: metadata.Source, Route: metadata.Route, Branch: branch, Guide: metadata.Guide, Goal: options.Goal, StartCommit: startCommit},
	})
}

func contributionStartCommit(root, branch string) string {
	for _, candidate := range [][]string{{"merge-base", "HEAD", "origin/" + branch}, {"merge-base", "HEAD", branch}, {"rev-parse", "HEAD"}} {
		if output, outputErr := exec.Command("git", append([]string{"-C", root}, candidate...)...).Output(); outputErr == nil {
			return strings.TrimSpace(string(output))
		}
	}
	return ""
}

type contributionCLIOptions struct {
	SiteURL   string
	Branch    string
	Checkout  string
	Host      string
	Port      int
	NoOpen    bool
	DraftFile string
	Goal      string
}

func parseContributionArgs(args []string) (contributionCLIOptions, error) {
	options := contributionCLIOptions{Host: "127.0.0.1"}
	for index := 0; index < len(args); index++ {
		argument := args[index]
		value := func(name string) (string, error) {
			if index+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			index++
			return args[index], nil
		}
		switch {
		case argument == "--branch":
			var err error
			options.Branch, err = value("--branch")
			if err != nil {
				return options, err
			}
		case strings.HasPrefix(argument, "--branch="):
			options.Branch = strings.TrimPrefix(argument, "--branch=")
		case argument == "--checkout":
			var err error
			options.Checkout, err = value("--checkout")
			if err != nil {
				return options, err
			}
		case strings.HasPrefix(argument, "--checkout="):
			options.Checkout = strings.TrimPrefix(argument, "--checkout=")
		case argument == "--host":
			var err error
			options.Host, err = value("--host")
			if err != nil {
				return options, err
			}
		case strings.HasPrefix(argument, "--host="):
			options.Host = strings.TrimPrefix(argument, "--host=")
		case argument == "--port":
			raw, err := value("--port")
			if err != nil {
				return options, err
			}
			options.Port, err = strconv.Atoi(raw)
			if err != nil || options.Port < 0 || options.Port > 65535 {
				return options, errors.New("--port must be between 0 and 65535")
			}
		case strings.HasPrefix(argument, "--port="):
			var err error
			options.Port, err = strconv.Atoi(strings.TrimPrefix(argument, "--port="))
			if err != nil || options.Port < 0 || options.Port > 65535 {
				return options, errors.New("--port must be between 0 and 65535")
			}
		case argument == "--no-open":
			options.NoOpen = true
		case argument == "--draft-file":
			var err error
			options.DraftFile, err = value("--draft-file")
			if err != nil {
				return options, err
			}
		case strings.HasPrefix(argument, "--draft-file="):
			options.DraftFile = strings.TrimPrefix(argument, "--draft-file=")
		case argument == "--goal":
			var err error
			options.Goal, err = value("--goal")
			if err != nil {
				return options, err
			}
		case strings.HasPrefix(argument, "--goal="):
			options.Goal = strings.TrimPrefix(argument, "--goal=")
		case strings.HasPrefix(argument, "-"):
			return options, fmt.Errorf("unknown contribute option %q", argument)
		case options.SiteURL == "":
			options.SiteURL = argument
		default:
			return options, errors.New("usage: mpress contribute <site-url-or-repository> [--goal page|translate] [--branch BRANCH] [--checkout DIRECTORY] [--port PORT] [--draft-file FILE] [--no-open]")
		}
	}
	if options.SiteURL == "" {
		return options, errors.New("usage: mpress contribute <site-url-or-repository> [--goal page|translate] [--branch BRANCH] [--checkout DIRECTORY] [--port PORT] [--draft-file FILE] [--no-open]")
	}
	if options.Goal != "" && options.Goal != "translate" && options.Goal != "page" {
		return options, errors.New("--goal must be translate or page")
	}
	return options, nil
}

func prepareDevProject(repository, checkout, branch string, positional []string) (string, error) {
	if repository != "" {
		if len(positional) != 0 {
			return "", errors.New("a project directory cannot be combined with --repo")
		}
		if checkout == "" {
			return "", errors.New("--checkout is required with --repo")
		}
		root, err := translate.CloneProject(context.Background(), repository, "", checkout)
		if err != nil {
			return "", err
		}
		if err := translate.PrepareWorkBranch(context.Background(), root, branch); err != nil {
			return "", err
		}
		fmt.Printf("Cloned %s to %s\n", repository, root)
		fmt.Printf("Prepared work branch %s\n", branch)
		return root, nil
	}
	if checkout != "" {
		return "", errors.New("--checkout requires --repo")
	}
	if len(positional) == 1 {
		return config.FindProject(positional[0])
	}
	return project()
}
func clean() error {
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	out := cfg.OutputPath(root)
	absRoot, _ := filepath.Abs(root)
	absOut, _ := filepath.Abs(out)
	rel, relErr := filepath.Rel(absRoot, absOut)
	if relErr != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("refusing unsafe clean target %s", out)
	}
	if err = os.RemoveAll(out); err == nil {
		fmt.Println("Cleaned", out)
	}
	return err
}
func checkSite() error {
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	items, err := check.Run(cfg.OutputPath(root))
	if err != nil {
		return err
	}
	if len(items) > 0 {
		fmt.Print(check.Format(items))
		return fmt.Errorf("%d broken link(s) or asset(s)", len(items))
	}
	fmt.Println("No broken links or assets")
	return nil
}
func importSite(args []string) error {
	var from, out, source string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		value := func(name string) (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			i++
			return args[i], nil
		}
		var err error
		switch {
		case arg == "--from":
			from, err = value("--from")
		case strings.HasPrefix(arg, "--from="):
			from = strings.TrimPrefix(arg, "--from=")
		case arg == "--output":
			out, err = value("--output")
		case strings.HasPrefix(arg, "--output="):
			out = strings.TrimPrefix(arg, "--output=")
		case strings.HasPrefix(arg, "-"):
			return fmt.Errorf("unknown import option %q", arg)
		case source == "":
			source = arg
		default:
			return errors.New("usage: mpress import --from starlight <source> --output <directory>")
		}
		if err != nil {
			return err
		}
	}
	if from != "starlight" {
		return fmt.Errorf("only --from starlight is supported in 0.1")
	}
	if source == "" {
		return errors.New("usage: mpress import --from starlight <source> --output <directory>")
	}
	if out == "" {
		return errors.New("--output is required")
	}
	return importer.ImportStarlight(source, out)
}

func convertSite(args []string) error {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	contentDirectory := fs.String("content", "", "content directory (defaults to build.contentDir)")
	targetFormat := fs.String("to", "mpd", "target document format: mpd or markdown")
	replace := fs.Bool("replace", false, "remove source files after every document converts successfully")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: mpress convert [--to mpd|markdown] [--content DIRECTORY] --replace")
	}
	if !*replace {
		return errors.New("mpress convert requires --replace to avoid duplicate routes")
	}
	format := strings.ToLower(strings.TrimSpace(*targetFormat))
	if format != "mpd" && format != "markdown" {
		return errors.New("--to must be mpd or markdown")
	}
	root, err := project()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	directory := strings.TrimSpace(*contentDirectory)
	if directory == "" {
		directory = cfg.ContentPath(root)
	} else if !filepath.IsAbs(directory) {
		directory = filepath.Join(root, filepath.FromSlash(directory))
	}
	result, err := projectconvert.Run(root, &cfg, directory, format)
	if err != nil {
		return err
	}
	fmt.Printf("Converted %d %s document(s) to %s in %s.\n", result.Count, result.SourceFormat, result.TargetFormat, result.Directory)
	return nil
}
func versions(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: mpress versions capture|list|verify|remove")
	}
	root, err := project()
	if err != nil {
		return err
	}
	switch args[0] {
	case "capture":
		fs := flag.NewFlagSet("capture", flag.ContinueOnError)
		force := fs.Bool("force", false, "replace existing")
		if err = fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("usage: mpress versions capture <label>")
		}
		err = docversion.Capture(root, fs.Arg(0), *force)
		if err == nil {
			fmt.Println("Captured", fs.Arg(0))
		}
		return err
	case "list":
		labels, err := docversion.List(root)
		if err != nil {
			return err
		}
		for _, x := range labels {
			fmt.Println(x)
		}
		return nil
	case "verify":
		if len(args) != 2 {
			return errors.New("usage: mpress versions verify <label>")
		}
		if err = docversion.Verify(root, args[1]); err == nil {
			fmt.Println("Verified", args[1])
		}
		return err
	case "remove":
		if len(args) != 2 {
			return errors.New("usage: mpress versions remove <label>")
		}
		return docversion.Remove(root, args[1])
	default:
		return fmt.Errorf("unknown versions command %q", args[0])
	}
}

const starterComponentsPage = `---
title: Try rich components
description: Explore every component included with M-Press.
order: 3
---

M-Press adds rich, accessible components to ordinary Markdown. This page uses every component included in the open-source renderer.

## Notes, details, badges, and buttons

@note{type="info" title="Markdown stays readable"}
Components use short directives. Their content remains ordinary Markdown.
@end

@details{title="Open this disclosure"}
Details can contain paragraphs, lists, links, and other Markdown content.

- They work without JavaScript.
- They use native browser semantics.
@end

@badge{type="default"}
Stable
@end

Use {badge:Preview} inside a sentence.

@button[Primary action](/getting-started/){primary|icon=play}
@button[Secondary action](/){secondary|icon=arrow-left}

@image{light="/images/component-light.svg" dark="/images/component-dark.svg" alt="Theme-aware documentation diagram"}

## Tabs

@tabs{sync-key="output"}
[Markdown]
Write pages in Markdown and keep them portable.

[Static output]
Publish generated HTML on any static host.

[No Node.js]
Build the complete site with one executable.
@end

## Terminal

@terminal{title="Build the site" frame="macos"}
$ mpress check
No issues found.
$ mpress build --strict
Built 3 pages.
@end

## Cards and link cards

@cards{cols="2"}
[Build your first site](/getting-started/)
Follow the guided tutorial.

---

[Return home](/)
Open the project onboarding page.
@end

@linkcard{title="Open the tutorial" href="/getting-started/" description="Make a change and see M-Press rebuild it." icon="arrow-right"}

## Steps

@steps
### Write
Add or edit a Markdown file.

### Preview
Save it and watch the browser update.

### Check
Run the built-in project checks.
@end

## File tree

@filetree
content/
  index.md  Project home
  getting-started.md  Guided tutorial
  components.md  Component catalogue
mpress.yaml  Complete site configuration
@end

## Diff

@diff{title="mpress.yaml" mode="inline"}
search:
  enabled: false
---
search:
  enabled: true
@end

## Explained code

@explained
~~~go
func main() { // (1)
    site.Build() // (2)
}
~~~

(1) Start with a regular Go entry point.

(2) Build the complete static site.
@end

## API endpoint

@api{method="POST" path="/v1/builds"}
Create a documentation build.

| Field | Type | Required |
| --- | --- | --- |
| ref | string | yes |
| strict | boolean | no |
@end

## API playground

@api-playground{method="GET" path="/search-index.json"}
Send a request to inspect this site's generated search index.
@end

## Comparison matrix

@matrix{highlight="M-Press"}
| Capability | M-Press | Typical Node SSG |
| --- | --- | --- |
| One executable | ✓ | ✗ |
| Rich components | ✓ | ✓ |
| Node.js runtime | ✗ | ✓ |
| Versions and translations | Built in | Add-ons |
@end

## Status

@status{service="Documentation build" state="operational"}
All systems normal
@end

@status{service="Translation queue" state="degraded"}
Jobs may take longer than usual
@end

## Calendar

@calendar{month="2026-08" style="compact"}
Use the controls to browse months.
@end

## Changelog

@changelog
### v0.2.0 (2026-08-03)
#### Added
- Complete starter component catalogue
- Scroll-aware table of contents

### v0.1.0 (2026-08-01)
#### Added
- Initial static site
@end

## Release notes

@release{version="0.2.0" date="2026-08-03" type="minor"}
### Highlights
- Every component is now visible in the starter project.
### New Features
- Interactive examples and layout helpers.
### Bug Fixes
- Stable scrolling above the developer bar.
@end

## Pricing

These fictional product plans demonstrate the pricing component.

@pricing{cols="2"}
### Personal
$0
[Start building](/getting-started/)
- ✓ One project
- ✓ Community support
---
### Team
$15/month
[Learn more](/)
recommended
- ✓ Shared projects
- ✓ Priority support
@end

## Testimonials

@testimonials{autoplay="0"}
“The content stays ordinary Markdown.”
Documentation author
---
“The output works without a client framework.”
Platform engineer
@end

## QR code

@qr{url="https://github.com/leaanthony/mpress" size="120" label="M-Press repository"}

## Guided tutorial

@tutorial{title="Publish this site"}
### Check the content
Run the project checks from the developer bar.

### Build strictly
Run mpress build with strict checking.

### Deploy the output
Publish the generated site directory.
@end

## Audience content

Use the audience selector to switch between these blocks.

@audience{role="developer"}
Developers can extend the Markdown processing pipeline in Go.
@end

@audience{role="writer"}
Writers can create the complete site with Markdown directives.
@end

## Conditional and variant content

@if{param="framework" value="plain" default="true"}
This is the default framework-neutral guidance.
@end

@if{param="framework" value="go"}
Add framework=go to the page URL to show this guidance.
@end

@variant{name="default"}
Build variants select product-specific content before publishing.
@end

## Reactive values

@input{name="projects" type="range" min="1" max="20" value="4" label="Projects"}

@input{name="editors" type="number" min="1" value="3" label="Editors per project"}

@computed{expr="projects * editors" deps="projects,editors" label="Project seats" format="%d"}

## Containers

@container{display=grid|columns=2|gap=1rem}
@tip[Static]
The complete layout exists in generated HTML.
@end
@info[Responsive]
Dense layouts collapse on smaller screens.
@end
@end

## Compact landing components

@preview-tabs
[Terminal]
Build and check the site from one executable.

[Components]
Add rich documentation without MDX.

[Output]
Publish static HTML anywhere.
@end

@callout{title="Built in"}
Search, navigation, versions, translations, and rich components ship together.
@end

@timeline
1. **Write** Create ordinary Markdown files.
2. **Preview** Save and see the browser update.
3. **Publish** Deploy the generated static directory.
@end

@capabilities
- **Search** Generate the search index at build time.
- **Translations** Publish language-aware routes.
- **Versions** Keep previous documentation releases available.
@end

@resources
[01 · Tutorial](/getting-started/)
## Build a site
Make your first working documentation change.
---
[02 · Configuration](/)
## Configure the project
Use the developer menu to edit every setting.
@end

## Documentation preview

@docs-preview{brand="Product docs"|search="Search documentation"|nav="START HERE:Overview*,Install;GUIDES:Configure,Deploy"|status="Built 3 pages · 0 errors"}
## Create your first project

Write Markdown, save the file, and publish static HTML.
@end

## Landing-page layout primitives

@section{aria-label="Landing component example"}
@columns
@column
@headline
Modern docs.
Just Markdown.
@end

Sections, columns, headlines, and action groups can build a complete landing page.

@actions
@button[Start now](/getting-started/){primary|icon=play}
@button[Go home](/){secondary|icon=arrow-left}
@end
@end

@column
Use layout components only when the content needs deliberate structure.
@end
@end
@end
`

const starterComponentLightSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 300" role="img" aria-labelledby="title"><title id="title">Documentation pipeline</title><rect width="960" height="300" rx="24" fill="#f4f6fa"/><g fill="none" stroke="#5375f6" stroke-width="4"><rect x="70" y="90" width="220" height="120" rx="14"/><rect x="370" y="90" width="220" height="120" rx="14"/><rect x="670" y="90" width="220" height="120" rx="14"/><path d="M290 150h80M590 150h80"/></g><g fill="#171a21" font-family="system-ui,sans-serif" font-size="25" font-weight="600" text-anchor="middle"><text x="180" y="160">Markdown</text><text x="480" y="160">M-Press</text><text x="780" y="160">Static site</text></g></svg>`

const starterComponentDarkSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 300" role="img" aria-labelledby="title"><title id="title">Documentation pipeline</title><rect width="960" height="300" rx="24" fill="#131925"/><g fill="none" stroke="#5375f6" stroke-width="4"><rect x="70" y="90" width="220" height="120" rx="14"/><rect x="370" y="90" width="220" height="120" rx="14"/><rect x="670" y="90" width="220" height="120" rx="14"/><path d="M290 150h80M590 150h80"/></g><g fill="#f0f2f5" font-family="system-ui,sans-serif" font-size="25" font-weight="600" text-anchor="middle"><text x="180" y="160">Markdown</text><text x="480" y="160">M-Press</text><text x="780" y="160">Static site</text></g></svg>`

func initProject(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "static"), 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"mpress.yaml": `site:
  title: "My Project"
  description: "Clear documentation written in Markdown"
  baseURL: ""
  defaultLanguage: en
  languages: [en]
  languageLabels:
    en: English
  defaultLanguageAtRoot: true
  missingTranslation: link-to-default
  logoLight: ""
  logoDark: ""
  favicon: ""
  socialImage: ""
  headerLinks: []
social:
  github: ""
  discord: ""
  reddit: ""
  x: ""
  rss: ""
  sponsor: ""
  editURL: ""
contribution:
  enabled: false
  repository: ""
  branch: main
  guide: CONTRIBUTING.md
  quickEdit: false
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
  customCSS: ""
theme:
  colorScheme: system
  accentColor: "#5375f6"
  hoverColor: "#7593ff"
  hoverColorLight: "#7593ff"
  hoverColorDark: "#7593ff"
  layout:
    preset: starlight
    contentWidth: 50rem
    wideContentWidth: 64rem
    sidebarWidth: 18.75rem
    tocWidth: 16rem
    contentTocGap: 2.5rem
    alignment: cluster
    toc: right
search:
  enabled: true
  shortcut: Mod+K
knowledge:
  enabled: true
accessibility:
  enabled: true
  shortcut: Mod+A
versioning:
  enabled: false
  current: ""
  artifactsDir: .mpress/versions
translation:
  provider: openrouter
  model: ""
  baseURL: https://openrouter.ai/api/v1
  apiKeyEnv: OPENROUTER_API_KEY
  sourceLanguage: en
  glossary: ""
  styleGuide: ""
  stateDir: .mpress/translations
  dataCollection: deny
  requireParameters: true
deploy:
  default: ""
  targets: {}
`,
		filepath.Join("content", "index.md"): `---
title: Start here
description: Learn how this M-Press project works.
---

This project is both a working documentation site and a short tutorial. Run ` + "`mpress dev`" + ` and use the M-Press menu in the status bar.

@button[Start project setup](#your-first-three-steps){primary|icon=play}

## Your first three steps

1. Open ` + "`content/getting-started.md`" + ` in your text editor.
2. Change a sentence and save the Markdown file.
3. M-Press rebuilds the site and reloads the browser.

@note{type="tip" title="Keep the source"}
Everything in this tutorial is ordinary Markdown. You can delete these pages when you are ready to write your own documentation.
@end

## What you get

- Rich documentation components without MDX
- Search, navigation, translations, and versioning
- A static output that does not need Node.js
`,
		filepath.Join("content", "getting-started.md"): `---
title: Build your first M-Press site
description: Make a real change and see it immediately.
order: 2
---

This tutorial gives you a working documentation site. You need Go 1.26.5 or later.

## Install M-Press

Run this command:

@terminal{title="Terminal"}
$ go install github.com/leaanthony/mpress/cmd/mpress@latest
@end

Confirm that the executable is available:

@terminal{title="Terminal"}
$ mpress version
@end

## Edit this page

Open this Markdown file in your text editor. Change this paragraph and save the file.

M-Press rebuilds the site and reloads the generated result.

## Add another page

Add a Markdown file to ` + "`content/`" + `. M-Press gives it a route and includes it in the next build.

## Run checks

Use **Run checks** in the status bar before you publish the site.
`,
		filepath.Join("content", "components.md"):                starterComponentsPage,
		filepath.Join("static", "images", "component-light.svg"): starterComponentLightSVG,
		filepath.Join("static", "images", "component-dark.svg"):  starterComponentDarkSVG,
		filepath.Join("content", "_nav.yaml"): `- label: Start here
  link: /
- label: Tutorial
  items:
    - label: Build your first site
      link: /getting-started/
    - label: Try rich components
      link: /components/
`,
		".gitignore":                           "site/\n.mpress/\n",
		filepath.Join(".mpress", "onboarding"): "Open the generated site to finish setup.\n",
	}
	for name, data := range files {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite %s", path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			return err
		}
	}
	fmt.Println("Created M-Press project in", dir)
	fmt.Print(initNextSteps(dir))
	return nil
}

func initNextSteps(dir string) string {
	quoted := "'" + strings.ReplaceAll(filepath.Clean(dir), "'", "'\"'\"'") + "'"
	return fmt.Sprintf("\nNext:\n  cd %s\n  mpress dev\n\nM-Press opens the project guide in your browser.\n", quoted)
}
