package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"

	"github.com/DarkInno/scion/internal/copytree"
	"github.com/DarkInno/scion/internal/registry"
)

type diffResult struct {
	Module             string   `json:"module"`
	Target             string   `json:"target"`
	ModuleVersion      string   `json:"moduleVersion"`
	RegistryVersion    string   `json:"registryVersion"`
	MetadataFound      bool     `json:"metadataFound"`
	MetadataStandalone bool     `json:"metadataStandalone"`
	MetadataWarnings   []string `json:"metadataWarnings,omitempty"`
	ModifiedFiles      []string `json:"modifiedFiles,omitempty"`
	MissingFiles       []string `json:"missingFiles,omitempty"`
	AddedFiles         []string `json:"addedFiles,omitempty"`
	UnchangedFiles     []string `json:"unchangedFiles,omitempty"`
	HasDifferences     bool     `json:"hasDifferences"`
}

func buildDiff(reg *registry.Bundle, module registry.Module, target string) (diffResult, error) {
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return diffResult{}, err
	}
	targetAbs = filepath.Clean(targetAbs)

	meta, found, err := readMetadata(targetAbs)
	if err != nil {
		return diffResult{}, err
	}

	standalone := false
	var warnings []string
	if found {
		standalone = meta.Standalone
		if meta.Module != "" && meta.Module != module.ID {
			warnings = append(warnings, "metadata module does not match requested module")
		}
		if meta.ModuleVersion != "" && meta.ModuleVersion != module.Version {
			warnings = append(warnings, "metadata module version differs from embedded registry")
		}
		if meta.RegistryVersion != "" && meta.RegistryVersion != reg.Index.Version {
			warnings = append(warnings, "metadata registry version differs from embedded registry")
		}
	} else {
		warnings = append(warnings, "metadata file not found")
	}

	files, err := reg.ModuleFiles(module, standalone)
	if err != nil {
		return diffResult{}, err
	}

	expected := make(map[string]registry.ModuleFile, len(files))
	for _, file := range files {
		expected[file.RelativePath] = file
	}

	result := diffResult{
		Module:             module.ID,
		Target:             targetAbs,
		ModuleVersion:      module.Version,
		RegistryVersion:    reg.Index.Version,
		MetadataFound:      found,
		MetadataStandalone: standalone,
		MetadataWarnings:   warnings,
	}

	for _, file := range files {
		targetFile, err := copytree.SafeJoin(targetAbs, file.RelativePath)
		if err != nil {
			return diffResult{}, err
		}
		data, err := os.ReadFile(targetFile)
		if os.IsNotExist(err) {
			result.MissingFiles = append(result.MissingFiles, file.RelativePath)
			continue
		}
		if err != nil {
			return diffResult{}, err
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != file.SHA256 {
			result.ModifiedFiles = append(result.ModifiedFiles, file.RelativePath)
		} else {
			result.UnchangedFiles = append(result.UnchangedFiles, file.RelativePath)
		}
	}

	if err := filepath.WalkDir(targetAbs, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(targetAbs, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == metadataFile {
			return nil
		}
		if _, ok := expected[rel]; !ok {
			result.AddedFiles = append(result.AddedFiles, rel)
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return diffResult{}, err
	}

	sort.Strings(result.ModifiedFiles)
	sort.Strings(result.MissingFiles)
	sort.Strings(result.AddedFiles)
	sort.Strings(result.UnchangedFiles)
	sort.Strings(result.MetadataWarnings)
	result.HasDifferences = len(result.ModifiedFiles) > 0 ||
		len(result.MissingFiles) > 0 ||
		len(result.AddedFiles) > 0 ||
		len(result.MetadataWarnings) > 0
	return result, nil
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
