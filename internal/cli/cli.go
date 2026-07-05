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
	case "-h", "--help", "help":
		writeUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		writeUsage(stderr)
		return 1
	}
}

func (a *App) runList(args []string, stdout, stderr io.Writer) int {
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
	return 0
}

func (a *App) runAdd(args []string, stdout, stderr io.Writer) int {
	moduleID, rest, err := takeFirstPositional(args)
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
	if strings.TrimSpace(*target) == "" {
		_, _ = fmt.Fprintf(stderr, "add requires --to <dir>\n")
		return 1
	}

	module, ok := a.reg.Index.Module(moduleID)
	if !ok {
		_, _ = fmt.Fprintf(stderr, "unknown module: %s\n", moduleID)
		return 1
	}
	if !module.StdlibOnly && !*standalone {
		_, _ = fmt.Fprintf(stderr, "module %s is marked stdlibOnly=false; use --standalone to copy its go.mod/go.sum or choose a zero-dependency module\n", module.ID)
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

	result, err := copytree.CopyFiles(*target, copyFiles, copytree.Options{DryRun: *dryRun, Force: *force})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "copy failed: %v\n", err)
		return 1
	}

	if *dryRun {
		_, _ = fmt.Fprintf(stdout, "Would copy %d files for %s to %s\n", len(result.Files), module.ID, *target)
	} else {
		_, _ = fmt.Fprintf(stdout, "Copied %s %s to %s\n", module.ID, module.Version, *target)
	}
	for _, file := range result.Files {
		_, _ = fmt.Fprintf(stdout, "  %s\n", file)
	}
	return 0
}

func (a *App) runDiff(args []string, stdout, stderr io.Writer) int {
	moduleID, rest, err := takeFirstPositional(args)
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
	if strings.TrimSpace(*target) == "" {
		_, _ = fmt.Fprintf(stderr, "diff requires --target <dir>\n")
		return 1
	}

	module, ok := a.reg.Index.Module(moduleID)
	if !ok {
		_, _ = fmt.Fprintf(stderr, "unknown module: %s\n", moduleID)
		return 1
	}
	result, err := buildDiff(a.reg, module, filepath.Clean(*target))
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

func takeFirstPositional(args []string) (string, []string, error) {
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		rest := make([]string, 0, len(args)-1)
		rest = append(rest, args[:i]...)
		rest = append(rest, args[i+1:]...)
		return arg, rest, nil
	}
	return "", nil, fmt.Errorf("missing positional argument")
}

func writeUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Usage:\n")
	_, _ = fmt.Fprintf(w, "  scion list [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion info <module> [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion add <module> --to <dir> [--dry-run] [--force] [--standalone]\n")
	_, _ = fmt.Fprintf(w, "  scion diff <module> --target <dir> [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion doctor [--strict] [--json]\n")
	_, _ = fmt.Fprintf(w, "  scion version [--json]\n")
}
