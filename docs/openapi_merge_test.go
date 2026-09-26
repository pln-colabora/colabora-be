package docs

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMergedOpenAPIContainsOnlyResolvableLocalComponentReferences(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { require.NoError(t, os.Chdir(workingDirectory)) })

	raw, err := buildMergedOpenAPI()
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &document))

	components, ok := document["components"].(map[string]any)
	require.True(t, ok)
	parameters, ok := components["parameters"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, parameters, "PermohonanID")
	paths, ok := document["paths"].(map[string]any)
	require.True(t, ok)
	for _, path := range []string{
		"/api/permohonan/{id}/wo-vendor/tiang",
		"/api/permohonan/{id}/wo-vendor/konstruksi",
		"/api/permohonan/{id}/wo-vendor/app",
		"/api/permohonan/{id}/reservasi-material",
		"/api/permohonan/{id}/wo-pdkb",
		"/api/permohonan/{id}/pelaksanaan-konstruksi",
		"/api/permohonan/{id}/pdkb-dokumentasi",
		"/api/permohonan/{id}/energize-jaringan",
		"/api/permohonan/{id}/pemasangan-sr-app",
		"/api/permohonan/{id}/closing",
		"/api/permohonan/{id}/vendor-assignments",
	} {
		require.Contains(t, paths, path)
	}
	assertResolvableReferences(t, document, document)
}

func TestPermohonanStageDocumentsKeepBusinessGrouping(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { require.NoError(t, os.Chdir(workingDirectory)) })

	expected := map[string]struct {
		tag   string
		paths []string
	}{
		"./docs/stage-01-permohonan.yaml": {
			tag: "Tahap 1 — Permohonan PB/PD",
			paths: []string{
				"/api/permohonan",
				"/api/tariffs",
				"/api/permohonan/{id}",
				"/api/permohonan/{id}/activities",
				"/api/permohonan/{id}/activities/{workflow_node}",
				"/api/permohonan/{id}/logs",
				"/api/permohonan/{id}/vendor-assignments",
			},
		},
		"./docs/stage-02-survei.yaml": {
			tag:   "Tahap 2 — Survei Perluasan Jaringan",
			paths: []string{"/api/permohonan/{id}/survei"},
		},
		"./docs/stage-03-perencanaan-perluasan.yaml": {
			tag:   "Tahap 3 — Perencanaan Perluasan",
			paths: []string{"/api/permohonan/{id}/rab-kko-kkf", "/api/permohonan/{id}/permohonan-perluasan"},
		},
		"./docs/stage-04-pra-pelaksanaan-konstruksi.yaml": {
			tag: "Tahap 4 — Pra Pelaksanaan Konstruksi",
			paths: []string{
				"/api/permohonan/{id}/wo-vendor/tiang",
				"/api/permohonan/{id}/wo-vendor/konstruksi",
				"/api/permohonan/{id}/wo-vendor/app",
				"/api/permohonan/{id}/reservasi-material",
				"/api/permohonan/{id}/tera-app",
				"/api/permohonan/{id}/wo-pdkb",
			},
		},
		"./docs/stage-05-pelaksanaan-konstruksi.yaml": {
			tag:   "Tahap 5 — Pelaksanaan Konstruksi",
			paths: []string{"/api/permohonan/{id}/pelaksanaan-konstruksi", "/api/permohonan/{id}/pdkb-dokumentasi"},
		},
		"./docs/stage-06-energize-jaringan.yaml": {
			tag:   "Tahap 6 — Energize Jaringan",
			paths: []string{"/api/permohonan/{id}/energize-jaringan", "/api/permohonan/{id}/pemasangan-sr-app"},
		},
		"./docs/stage-07-penutupan.yaml": {
			tag:   "Tahap 7 — Penutupan / Selesai",
			paths: []string{"/api/permohonan/{id}/closing"},
		},
	}

	for file, want := range expected {
		raw, err := os.ReadFile(file)
		require.NoError(t, err, file)
		var document map[string]any
		require.NoError(t, yaml.Unmarshal(raw, &document), file)

		tags, ok := document["tags"].([]any)
		require.True(t, ok, file)
		require.Len(t, tags, 1, file)
		tag, ok := tags[0].(map[string]any)
		require.True(t, ok, file)
		require.Equal(t, want.tag, tag["name"], file)

		paths, ok := document["paths"].(map[string]any)
		require.True(t, ok, file)
		require.Len(t, paths, len(want.paths), file)
		for _, path := range want.paths {
			require.Contains(t, paths, path, file)
		}
	}
}

func TestMergedOpenAPIIncludesCurrentRequestExamples(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(".."))
	t.Cleanup(func() { require.NoError(t, os.Chdir(workingDirectory)) })

	raw, err := buildMergedOpenAPI()
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &document))

	// These values exercise the current public contract: registration requires a
	// verification document, construction WO requires an estimated finish date,
	// and vendor-owned location-bearing activities expose coordinates.
	serialized := string(raw)
	require.Contains(t, serialized, "budi.santoso@example.test")
	require.Contains(t, serialized, "surat-verifikasi-vendor.pdf")
	require.Contains(t, serialized, "estimasi_tanggal_selesai")
	require.Contains(t, serialized, "location_coordinates")
	require.Contains(t, serialized, "-6.2")
	require.Contains(t, serialized, "106.816666")
	require.Contains(t, serialized, "550e8400-e29b-41d4-a716-446655440005")
}

func assertResolvableReferences(t *testing.T, root map[string]any, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "$ref" {
				reference, ok := item.(string)
				require.True(t, ok)
				if strings.HasPrefix(reference, "#/components/") {
					require.True(t, resolvesLocalReference(root, reference), "unresolved OpenAPI reference: %s", reference)
				}
				continue
			}
			assertResolvableReferences(t, root, item)
		}
	case []any:
		for _, item := range typed {
			assertResolvableReferences(t, root, item)
		}
	}
}

func resolvesLocalReference(root map[string]any, reference string) bool {
	var current any = root
	for _, segment := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
		mapping, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = mapping[segment]
		if !ok {
			return false
		}
	}
	return true
}
