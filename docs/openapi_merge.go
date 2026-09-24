package docs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// moduleDocFiles lists the self-contained OpenAPI documents that get combined
// into the single reference served at /docs/openapi.yaml. Permohonan is split
// into seven business-process stage documents so Scalar can show the same
// grouping used by the workflow UI.
var moduleDocFiles = []string{
	"./docs/health.yaml",
	"./docs/auth.yaml",
	"./docs/user.yaml",
	"./docs/stage-01-permohonan.yaml",
	"./docs/stage-02-survei.yaml",
	"./docs/stage-03-perencanaan-perluasan.yaml",
	"./docs/stage-04-pra-pelaksanaan-konstruksi.yaml",
	"./docs/stage-05-pelaksanaan-konstruksi.yaml",
	"./docs/stage-06-energize-jaringan.yaml",
	"./docs/stage-07-penutupan.yaml",
	"./docs/document.yaml",
}

// buildMergedOpenAPI combines the per-module OpenAPI documents into a single
// document so Scalar renders every module as one flat, always-visible list of
// sidebar sections instead of a dropdown you have to switch between. Stage
// documents share identical copies of common response and workflow schemas by
// convention, so last-write-wins merging of components is safe as long as
// those shared definitions remain compatible.
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
	parameters := map[string]any{}

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
			mergeMapInto(parameters, comp["parameters"])
		}
	}

	merged["servers"] = servers
	merged["tags"] = tags
	merged["paths"] = paths
	merged["components"] = map[string]any{
		"schemas":         schemas,
		"responses":       responses,
		"securitySchemes": securitySchemes,
		"parameters":      parameters,
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
