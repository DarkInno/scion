package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

const IndexSchemaVersion = 1

type Index struct {
	SchemaVersion int      `json:"schemaVersion"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	Patterns      []Module `json:"patterns"`
}

type Module struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Languages         []string `json:"languages"`
	Frameworks        []string `json:"frameworks"`
	Tags              []string `json:"tags"`
	Version           string   `json:"version"`
	Package           string   `json:"package"`
	Source            string   `json:"source"`
	DefaultTarget     string   `json:"defaultTarget"`
	StdlibOnly        bool     `json:"stdlibOnly"`
	Include           []string `json:"include"`
	StandaloneInclude []string `json:"standaloneInclude"`
	Status            string   `json:"status"`
}

func ParseIndex(data []byte) (*Index, error) {
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	if idx.SchemaVersion != IndexSchemaVersion {
		return nil, fmt.Errorf("unsupported registry index schema %d", idx.SchemaVersion)
	}
	if len(idx.Patterns) == 0 {
		return nil, errors.New("registry index has no modules")
	}
	return &idx, nil
}

func (idx *Index) Module(id string) (Module, bool) {
	for _, module := range idx.Patterns {
		if module.ID == id {
			return module, true
		}
	}
	return Module{}, false
}

func (idx *Index) SortedModules() []Module {
	modules := append([]Module(nil), idx.Patterns...)
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].ID < modules[j].ID
	})
	return modules
}
