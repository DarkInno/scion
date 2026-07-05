package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const metadataFile = ".scion-module.json"

type moduleMetadata struct {
	SchemaVersion   int               `json:"schemaVersion"`
	Module          string            `json:"module"`
	ModuleVersion   string            `json:"moduleVersion"`
	RegistryVersion string            `json:"registryVersion"`
	CopiedFiles     []string          `json:"copiedFiles"`
	SourceHashes    map[string]string `json:"sourceHashes"`
	Standalone      bool              `json:"standalone"`
}

func readMetadata(root string) (moduleMetadata, bool, error) {
	data, err := os.ReadFile(filepath.Join(root, metadataFile))
	if os.IsNotExist(err) {
		return moduleMetadata{}, false, nil
	}
	if err != nil {
		return moduleMetadata{}, false, err
	}
	var meta moduleMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return moduleMetadata{}, false, err
	}
	return meta, true, nil
}
