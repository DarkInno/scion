package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/DarkInno/scion/internal/copytree"
	"github.com/DarkInno/scion/internal/doctor"
	"github.com/DarkInno/scion/internal/registry"
	"github.com/DarkInno/scion/internal/version"
)

type App struct {
	reg *registry.Bundle
}

func New(reg *registry.Bundle) *App {
	return &App{reg: reg}
}

func (a *App) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	_ = ctx
	if len(args) == 0 {
		writeUsage(stderr)
		return 1
	}

	switch args[0] {
	case "list":
		return a.runList(args[1:], stdout, stderr)
	case "info":
		return a.runInfo(args[1:], stdout, stderr)
	case "add":
		return a.runAdd(args[1:], stdout, stderr)
	case "diff":
		return a.runDiff(args[1:], stdout, stderr)
	case "doctor":
		return a.runDoctor(args[1:], stdout, stderr)
	case "version":
		return a.runVersion(args[1:], stdout, stderr)
	case "-h", "--help":
		writeUsage(stdout)
		return 0
	case "help":
		if len(args) == 1 {
			writeUsage(stdout)
			return 0
		}
		if writeCommandUsage(stdout, args[1]) {
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "unknown help topic: %s\n\n", args[1])
		writeUsage(stderr)
		return 1
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		writeUsage(stderr)
		return 1
	}
}

func (a *App) runList(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		writeCommandUsage(stdout, "list")
		return 0
	}
	fs := newFlagSet("list", stderr)
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "list does not accept positional arguments\n")
		return 1
	}

	modules := a.reg.Index.SortedModules()
	if *jsonOut {
		_ = json.NewEncoder(stdout).Encode(modules)
		return 0
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(tw, "ID\tPACKAGE\tSTATUS\tSTDLIB\tDEFAULT TARGET\n")
	for _, module := range modules {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%t\t%s\n", module.ID, module.Package, module.Status, module.StdlibOnly, module.DefaultTarget)
	}
	_ = tw.Flush()
	return 0
}

func (a *App) runInfo(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		writeCommandUsage(stdout, "info")
		return 0
	}
	moduleID, rest, err := takeFirstPositional(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "info requires a module id\n")
		return 1
	}
	fs := newFlagSet("info", stderr)
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(rest); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected argument: %s\n", fs.Arg(0))
		return 1
	}

	module, ok := a.reg.Index.Module(moduleID)
	if !ok {
		_, _ = fmt.Fprintf(stderr, "unknown module: %s\n", moduleID)
		return 1
	}
	if *jsonOut {
		_ = json.NewEncoder(stdout).Encode(module)
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "ID: %s\n", module.ID)
	_, _ = fmt.Fprintf(stdout, "Name: %s\n", module.Name)
	_, _ = fmt.Fprintf(stdout, "Description: %s\n", module.Description)
	_, _ = fmt.Fprintf(stdout, "Version: %s\n", module.Version)
	_, _ = fmt.Fprintf(stdout, "Package: %s\n", module.Package)
	_, _ = fmt.Fprintf(stdout, "Source: %s\n", module.Source)
	_, _ = fmt.Fprintf(stdout, "Default target: %s\n", module.DefaultTarget)
	_, _ = fmt.Fprintf(stdout, "Standard library only: %t\n", module.StdlibOnly)
	_, _ = fmt.Fprintf(stdout, "Status: %s\n", module.Status)
	if !module.StdlibOnly {
		_, _ = fmt.Fprintf(stdout, "Standalone required: true\n")
		_, _ = fmt.Fprintf(stdout, "Add command: scion add %s --standalone\n", module.ID)
	}
	return 0
}

func (a *App) runAdd(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		writeCommandUsage(stdout, "add")
		return 0
	}
	moduleID, rest, err := takeFirstPositional(args, "to")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add requires a module id\n")
		return 1
	}
	fs := newFlagSet("add", stderr)
	target := fs.String("to", "", "destination directory")
	dryRun := fs.Bool("dry-run", false, "show files without writing")
	force := fs.Bool("force", false, "overwrite files copied by the bundle")
	standalone := fs.Bool("standalone", false, "copy go.mod/go.sum as a standalone module")
	if err := fs.Parse(rest); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected argument: %s\n", fs.Arg(0))
		return 1
	}

	module, ok := a.reg.Index.Module(moduleID)
	if !ok {
		_, _ = fmt.Fprintf(stderr, "unknown module: %s\n", moduleID)
		return 1
	}
	targetDir := strings.TrimSpace(*target)
	if targetDir == "" {
		targetDir = module.DefaultTarget
	}
	if targetDir == "" {
		_, _ = fmt.Fprintf(stderr, "add requires --to <dir>; module %s has no default target\n", module.ID)
		return 1
	}
	if !module.StdlibOnly && !*standalone {
		_, _ = fmt.Fprintf(stderr, "module %s is marked stdlibOnly=false; use --standalone to copy its go.mod/go.sum explicitly\n", module.ID)
		return 1
	}

	files, err := a.reg.ModuleFiles(module, *standalone)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read module files: %v\n", err)
		return 1
	}

	copyFiles := make([]copytree.File, 0, len(files)+1)
	sourceHashes := make(map[string]string, len(files))
	copied := make([]string, 0, len(files))
	for _, file := range files {
		data, err := a.reg.ReadFile(file.SourcePath)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "read %s: %v\n", file.SourcePath, err)
			return 1
		}
		copyFiles = append(copyFiles, copytree.File{RelPath: file.RelativePath, Data: data, Mode: 0o644})
		sourceHashes[file.RelativePath] = file.SHA256
		copied = append(copied, file.RelativePath)
	}
	sort.Strings(copied)
	meta := moduleMetadata{
		SchemaVersion:   1,
		Module:          module.ID,
		ModuleVersion:   module.Version,
		RegistryVersion: a.reg.Index.Version,
		CopiedFiles:     copied,
		SourceHashes:    sourceHashes,
		Standalone:      *standalone,
	}
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "build metadata: %v\n", err)
		return 1
	}
	metaData = append(metaData, '\n')
	copyFiles = append(copyFiles, copytree.File{RelPath: metadataFile, Data: metaData, Mode: 0o644})

	result, err := copytree.CopyFiles(targetDir, copyFiles, copytree.Options{DryRun: *dryRun, Force: *force})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "copy failed: %v\n", err)
		return 1
	}

	if *dryRun {
		_, _ = fmt.Fprintf(stdout, "Would copy %d files for %s to %s\n", len(result.Files), module.ID, targetDir)
	} else {
		_, _ = fmt.Fprintf(stdout, "Copied %s %s to %s\n", module.ID, module.Version, targetDir)
	}
	for _, file := range result.Files {
		_, _ = fmt.Fprintf(stdout, "  %s\n", file)
	}
	return 0
}

func (a *App) runDiff(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		writeCommandUsage(stdout, "diff")
		return 0
	}
	moduleID, rest, err := takeFirstPositional(args, "target")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "diff requires a module id\n")
		return 1
	}
	fs := newFlagSet("diff", stderr)
	target := fs.String("target", "", "destination directory")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(rest); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected argument: %s\n", fs.Arg(0))
		return 1
	}

	module, ok := a.reg.Index.Module(moduleID)
	if !ok {
		_, _ = fmt.Fprintf(stderr, "unknown module: %s\n", moduleID)
		return 1
	}
	targetDir := strings.TrimSpace(*target)
	if targetDir == "" {
		targetDir = module.DefaultTarget
	}
	if targetDir == "" {
		_, _ = fmt.Fprintf(stderr, "diff requires --target <dir>; module %s has no default target\n", module.ID)
		return 1
	}
	result, err := buildDiff(a.reg, module, filepath.Clean(targetDir))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "diff failed: %v\n", err)
		return 1
	}
	if *jsonOut {
		_ = json.NewEncoder(stdout).Encode(result)
	} else {
		writeDiff(stdout, result)
	}
	if result.HasDifferences {
		return 2
	}
	return 0
}

func (a *App) runDoctor(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		writeCommandUsage(stdout, "doctor")
		return 0
	}
	fs := newFlagSet("doctor", stderr)
	strict := fs.Bool("strict", false, "treat release-gate warnings as errors")
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected argument: %s\n", fs.Arg(0))
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "doctor failed: %v\n", err)
		return 1
	}
	report := doctor.Run(a.reg, cwd, *strict)
	if *jsonOut {
		_ = json.NewEncoder(stdout).Encode(report)
	} else {
		writeDoctor(stdout, report)
	}
	if len(report.Issues) > 0 {
		return 2
	}
	return 0
}

func (a *App) runVersion(args []string, stdout, stderr io.Writer) int {
	if hasHelp(args) {
		writeCommandUsage(stdout, "version")
		return 0
	}
	fs := newFlagSet("version", stderr)
	jsonOut := fs.Bool("json", false, "write JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected argument: %s\n", fs.Arg(0))
		return 1
	}

	currentVersion := version.Current()
	currentCommit := version.CurrentCommit()
	out := map[string]string{
		"version":         currentVersion,
		"commit":          currentCommit,
		"registryVersion": a.reg.Index.Version,
		"bundleHash":      a.reg.Manifest.BundleHash,
	}
	if *jsonOut {
		_ = json.NewEncoder(stdout).Encode(out)
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "scion %s\n", currentVersion)
	_, _ = fmt.Fprintf(stdout, "commit: %s\n", currentCommit)
	_, _ = fmt.Fprintf(stdout, "registry: %s\n", a.reg.Index.Version)
	_, _ = fmt.Fprintf(stdout, "bundle: %s\n", a.reg.Manifest.BundleHash)
	return 0
}

func writeDiff(w io.Writer, result diffResult) {
	if !result.HasDifferences {
		_, _ = fmt.Fprintf(w, "%s matches embedded registry version %s\n", result.Module, result.ModuleVersion)
		return
	}
	for _, warning := range result.MetadataWarnings {
		_, _ = fmt.Fprintf(w, "warning: %s\n", warning)
	}
	writeFileGroup(w, "modified", result.ModifiedFiles)
	writeFileGroup(w, "missing", result.MissingFiles)
	writeFileGroup(w, "added", result.AddedFiles)
	_, _ = fmt.Fprintf(w, "%d modified, %d missing, %d added file%s\n",
		len(result.ModifiedFiles),
		len(result.MissingFiles),
		len(result.AddedFiles),
		pluralSuffix(len(result.ModifiedFiles)+len(result.MissingFiles)+len(result.AddedFiles)),
	)
}

func writeFileGroup(w io.Writer, label string, files []string) {
	if len(files) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "%s:\n", label)
	for _, file := range files {
		_, _ = fmt.Fprintf(w, "  %s\n", file)
	}
}

func writeDoctor(w io.Writer, report doctor.Report) {
	if len(report.Issues) == 0 {
		_, _ = fmt.Fprintf(w, "doctor ok\n")
		return
	}
	for _, issue := range report.Issues {
		if issue.Module != "" {
			_, _ = fmt.Fprintf(w, "%s [%s]: %s\n", issue.Level, issue.Module, issue.Message)
		} else {
			_, _ = fmt.Fprintf(w, "%s: %s\n", issue.Level, issue.Message)
		}
	}
	if report.OK {
		_, _ = fmt.Fprintf(w, "doctor completed with warnings\n")
	} else {
		_, _ = fmt.Fprintf(w, "doctor found errors\n")
	}
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func hasHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return true
		}
	}
	return false
}

func takeFirstPositional(args []string, valueFlags ...string) (string, []string, error) {
	flagsWithValues := make(map[string]bool, len(valueFlags))
	for _, flagName := range valueFlags {
		flagsWithValues[flagName] = true
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				return args[i+1], removeArgAt(args, i+1), nil
			}
			break
		}
		if strings.HasPrefix(arg, "-") {
			name := strings.TrimLeft(arg, "-")
			if cut, _, ok := strings.Cut(name, "="); ok {
				name = cut
			}
			if flagsWithValues[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
			}
			continue
		}
		return args[i], removeArgAt(args, i), nil
	}
	return "", nil, fmt.Errorf("missing positional argument")
}

func removeArgAt(args []string, index int) []string {
	rest := make([]string, 0, len(args)-1)
	rest = append(rest, args[:index]...)
	rest = append(rest, args[index+1:]...)
	return rest
}

func writeUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Usage:\n")
	_, _ = fmt.Fprintf(w, "  scion list [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion info <module> [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion add <module> [--to <dir>] [--dry-run] [--force] [--standalone]\n")
	_, _ = fmt.Fprintf(w, "  scion diff <module> [--target <dir>] [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion doctor [--strict] [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion version [--json]\n")
	_, _ = fmt.Fprintf(w, "\nRun \"scion help <command>\" for command details.\n")
}

func writeCommandUsage(w io.Writer, command string) bool {
	switch command {
	case "list":
		_, _ = fmt.Fprintf(w, "Usage: scion list [--json]\n\n")
		_, _ = fmt.Fprintf(w, "List embedded source templates.\n\n")
		_, _ = fmt.Fprintf(w, "Options:\n")
		_, _ = fmt.Fprintf(w, "  --json  write JSON\n")
	case "info":
		_, _ = fmt.Fprintf(w, "Usage: scion info <module> [--json]\n\n")
		_, _ = fmt.Fprintf(w, "Show metadata for one embedded source template.\n\n")
		_, _ = fmt.Fprintf(w, "Examples:\n")
		_, _ = fmt.Fprintf(w, "  scion info cache\n")
		_, _ = fmt.Fprintf(w, "  scion info auth --json\n\n")
		_, _ = fmt.Fprintf(w, "Options:\n")
		_, _ = fmt.Fprintf(w, "  --json  write JSON\n")
	case "add":
		_, _ = fmt.Fprintf(w, "Usage: scion add <module> [--to <dir>] [--dry-run] [--force] [--standalone]\n\n")
		_, _ = fmt.Fprintf(w, "Copy a source template into your project. If --to is omitted, Scion uses the module default target.\n\n")
		_, _ = fmt.Fprintf(w, "Examples:\n")
		_, _ = fmt.Fprintf(w, "  scion add cache --dry-run\n")
		_, _ = fmt.Fprintf(w, "  scion add cache\n")
		_, _ = fmt.Fprintf(w, "  scion add auth --standalone\n\n")
		_, _ = fmt.Fprintf(w, "Options:\n")
		_, _ = fmt.Fprintf(w, "  --to <dir>      destination directory; defaults to the module default target\n")
		_, _ = fmt.Fprintf(w, "  --dry-run       show files without writing\n")
		_, _ = fmt.Fprintf(w, "  --force         overwrite files copied by the bundle\n")
		_, _ = fmt.Fprintf(w, "  --standalone    copy go.mod/go.sum for modules with external dependencies\n")
	case "diff":
		_, _ = fmt.Fprintf(w, "Usage: scion diff <module> [--target <dir>] [--json]\n\n")
		_, _ = fmt.Fprintf(w, "Compare a copied module with the embedded source template. If --target is omitted, Scion uses the module default target.\n\n")
		_, _ = fmt.Fprintf(w, "Examples:\n")
		_, _ = fmt.Fprintf(w, "  scion diff cache\n")
		_, _ = fmt.Fprintf(w, "  scion diff cache --target internal/cache\n\n")
		_, _ = fmt.Fprintf(w, "Options:\n")
		_, _ = fmt.Fprintf(w, "  --target <dir>  destination directory; defaults to the module default target\n")
		_, _ = fmt.Fprintf(w, "  --json          write JSON\n")
	case "doctor":
		_, _ = fmt.Fprintf(w, "Usage: scion doctor [--strict] [--json]\n\n")
		_, _ = fmt.Fprintf(w, "Check registry, bundle, and module release readiness.\n\n")
		_, _ = fmt.Fprintf(w, "Options:\n")
		_, _ = fmt.Fprintf(w, "  --strict  treat release-gate warnings as errors\n")
		_, _ = fmt.Fprintf(w, "  --json    write JSON\n")
	case "version":
		_, _ = fmt.Fprintf(w, "Usage: scion version [--json]\n\n")
		_, _ = fmt.Fprintf(w, "Print CLI version, commit, registry version, and bundle hash.\n\n")
		_, _ = fmt.Fprintf(w, "Options:\n")
		_, _ = fmt.Fprintf(w, "  --json  write JSON\n")
	default:
		return false
	}
	return true
}
