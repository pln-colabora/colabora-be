package docs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// moduleDocFiles lists the self-contained per-module OpenAPI documents that
// get combined into the single reference served at /docs/openapi.yaml. Adding
// a new module's docs: add its `docs/<module>.yaml` file path here (plus the
// `server.StaticFile` line below if you still want it downloadable on its
// own) — no other wiring needed for it to show up in the combined reference.
var moduleDocFiles = []string{
	"./docs/health.yaml",
	"./docs/auth.yaml",
	"./docs/user.yaml",
	"./docs/permohonan.yaml",
}

// buildMergedOpenAPI combines the per-module OpenAPI documents into a single
// document so Scalar renders every module as one flat, always-visible list of
// sidebar sections instead of a dropdown you have to switch between. Modules
// share identical copies of a few boilerplate components (ApiResponse,
// BadRequest, Unauthorized, bearerAuth, UserResponse) by convention — see
// CLAUDE.md — so last-write-wins merging of components is safe as long as
// that convention holds.
func buildMergedOpenAPI() ([]byte, error) {
	merged := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "COLABORA API",
			"description": "Full API reference, combining every module's OpenAPI document.",
			"version":     "1.0.0",
		},
	}

	var servers any
	tagsSeen := map[string]bool{}
	var tags []any
	paths := map[string]any{}
	schemas := map[string]any{}
	responses := map[string]any{}
	securitySchemes := map[string]any{}

	for _, file := range moduleDocFiles {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}

		var doc map[string]any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", file, err)
		}

		if servers == nil {
			servers = doc["servers"]
		}

		if docTags, ok := doc["tags"].([]any); ok {
			for _, t := range docTags {
				if tagMap, ok := t.(map[string]any); ok {
					if name, _ := tagMap["name"].(string); name != "" && !tagsSeen[name] {
						tagsSeen[name] = true
						tags = append(tags, t)
					}
				}
			}
		}

		if docPaths, ok := doc["paths"].(map[string]any); ok {
			for k, v := range docPaths {
				paths[k] = v
			}
		}

		if comp, ok := doc["components"].(map[string]any); ok {
			mergeMapInto(schemas, comp["schemas"])
			mergeMapInto(responses, comp["responses"])
			mergeMapInto(securitySchemes, comp["securitySchemes"])
		}
	}

	merged["servers"] = servers
	merged["tags"] = tags
	merged["paths"] = paths
	merged["components"] = map[string]any{
		"schemas":         schemas,
		"responses":       responses,
		"securitySchemes": securitySchemes,
	}

	return yaml.Marshal(merged)
}

func mergeMapInto(dst map[string]any, src any) {
	srcMap, ok := src.(map[string]any)
	if !ok {
		return
	}
	for k, v := range srcMap {
		dst[k] = v
	}
}
