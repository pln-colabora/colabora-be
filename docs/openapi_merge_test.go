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
