package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

var architectureDiagramFiles = []string{
	"01-workflow-jtr-jtm.md",
	"02-workflow-plg-tm.md",
	"03-workflow-state.md",
	"04-rbac-ownership.md",
	"05-activity-submission.md",
	"06-data-model.md",
	"07-components.md",
	"08-api-action-map.md",
	"09-evidence-lifecycle.md",
	"10-sla-critical-path.md",
}

func TestArchitectureDocumentationRoutes(t *testing.T) {
	withRepositoryRoot(t)
	gin.SetMode(gin.TestMode)
	server := gin.New()
	RegisterRoutes(server)

	paths := []string{"/docs/architecture"}
	for _, name := range architectureDiagramFiles {
		paths = append(paths, "/docs/architecture/diagrams/"+name)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code)
			require.NotEmpty(t, response.Body.String())
		})
	}

	viewerRequest := httptest.NewRequest(http.MethodGet, "/docs/architecture", nil)
	viewerResponse := httptest.NewRecorder()
	server.ServeHTTP(viewerResponse, viewerRequest)
	for _, name := range architectureDiagramFiles {
		require.Contains(t, viewerResponse.Body.String(), name)
	}

	referenceRequest := httptest.NewRequest(http.MethodGet, "/docs", nil)
	referenceResponse := httptest.NewRecorder()
	server.ServeHTTP(referenceResponse, referenceRequest)
	require.Contains(t, referenceResponse.Body.String(), `href="/docs/architecture"`)
}

func TestArchitectureDiagramSourcesHaveOneMermaidBlock(t *testing.T) {
	withRepositoryRoot(t)
	for _, name := range architectureDiagramFiles {
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("docs", "architecture", "diagrams", name))
			require.NoError(t, err)
			source := string(content)
			require.Equal(t, 1, strings.Count(source, "```mermaid"))
			start := strings.Index(source, "```mermaid") + len("```mermaid")
			end := strings.Index(source[start:], "```")
			require.Greater(t, end, 0)
			definition := strings.TrimSpace(source[start : start+end])
			require.True(t,
				strings.HasPrefix(definition, "flowchart ") ||
					strings.HasPrefix(definition, "stateDiagram-v2") ||
					strings.HasPrefix(definition, "sequenceDiagram") ||
					strings.HasPrefix(definition, "erDiagram"),
				"unsupported Mermaid diagram declaration in %s", name,
			)
		})
	}
}

func TestArchitectureWorkflowDiagramsContainEveryCanonicalNode(t *testing.T) {
	withRepositoryRoot(t)
	for _, name := range architectureDiagramFiles[:2] {
		content, err := os.ReadFile(filepath.Join("docs", "architecture", "diagrams", name))
		require.NoError(t, err)
		for _, definition := range workflow.Definitions() {
			require.Contains(t, string(content), string(definition.Code), "%s is missing from %s", definition.Code, name)
		}
	}
}

func withRepositoryRoot(t *testing.T) {
	t.Helper()
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	if filepath.Base(workingDirectory) == "docs" {
		require.NoError(t, os.Chdir(".."))
		t.Cleanup(func() { require.NoError(t, os.Chdir(workingDirectory)) })
	}
}
