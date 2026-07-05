package doctor_test

import (
	"testing"

	"github.com/DarkInno/scion/internal/bundle"
	"github.com/DarkInno/scion/internal/doctor"
	"github.com/DarkInno/scion/internal/registry"
)

func TestDoctorStrictPassesReleaseReadyRegistry(t *testing.T) {
	reg, err := registry.NewBundle(bundle.RegistryZip, bundle.ManifestJSON)
	if err != nil {
		t.Fatalf("load embedded bundle: %v", err)
	}

	report := doctor.Run(reg, t.TempDir(), false)
	if !report.OK || len(report.Issues) != 0 {
		t.Fatalf("non-strict doctor should pass: %+v", report.Issues)
	}
	report = doctor.Run(reg, t.TempDir(), true)
	if !report.OK || len(report.Issues) != 0 {
		t.Fatalf("strict doctor should pass: %+v", report.Issues)
	}
}
