package registry

type Manifest struct {
	SchemaVersion   int            `json:"schemaVersion"`
	RegistryVersion string         `json:"registryVersion"`
	BundleHash      string         `json:"bundleHash"`
	Modules         []BundleModule `json:"modules"`
	Files           []BundleFile   `json:"files"`
}

type BundleModule struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

type BundleFile struct {
	Path          string `json:"path"`
	Module        string `json:"module,omitempty"`
	ModuleVersion string `json:"moduleVersion,omitempty"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
}
