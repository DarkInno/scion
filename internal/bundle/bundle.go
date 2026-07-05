package bundle

import _ "embed"

//go:embed registry.zip
var RegistryZip []byte

//go:embed manifest.json
var ManifestJSON []byte
