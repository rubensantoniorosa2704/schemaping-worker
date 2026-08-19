package diff

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
	"gopkg.in/yaml.v3"
)

// OpenAPI implements domain.DiffStrategy for OpenAPI 3.0 specification files.
// It compares the spec structure between two snapshots using the raw response bytes.
//
// Limitations:
//   - $ref references are NOT resolved. Only inline definitions are compared.
//   - Only top-level paths, operations, parameters, and response codes are tracked.
//   - Deep schema diffing within components is not performed.
type OpenAPI struct{}

// Diff compares two OpenAPI specs (from RawBody) and returns detected changes.
func (s OpenAPI) Diff(prev, curr domain.Snapshot) []domain.DiffResult {
	prevSpec, errP := parseSpec(prev.RawBody)
	currSpec, errC := parseSpec(curr.RawBody)

	if errP != nil || errC != nil {
		return nil
	}

	var results []domain.DiffResult

	// Compare info.version
	if prevSpec.Info.Version != currSpec.Info.Version {
		results = append(results, domain.DiffResult{
			Kind:   domain.ChangeKindTypeChanged,
			Path:   "info.version",
			Before: prevSpec.Info.Version,
			After:  currSpec.Info.Version,
		})
	}

	results = append(results, diffPaths(prevSpec.Paths, currSpec.Paths)...)

	return results
}

// --- internal OpenAPI types (minimal, no $ref resolution) ---

type openAPISpec struct {
	Info  openAPIInfo                      `yaml:"info" json:"info"`
	Paths map[string]map[string]operation  `yaml:"paths" json:"paths"`
}

type openAPIInfo struct {
	Version string `yaml:"version" json:"version"`
}

type operation struct {
	OperationID string              `yaml:"operationId" json:"operationId"`
	Parameters  []parameter         `yaml:"parameters" json:"parameters"`
	RequestBody *requestBody        `yaml:"requestBody" json:"requestBody"`
	Responses   map[string]response `yaml:"responses" json:"responses"`
}

type parameter struct {
	Name     string `yaml:"name" json:"name"`
	In       string `yaml:"in" json:"in"`
	Required bool   `yaml:"required" json:"required"`
	Schema   *schemaObj `yaml:"schema" json:"schema"`
}

type requestBody struct {
	Required bool                   `yaml:"required" json:"required"`
	Content  map[string]mediaType   `yaml:"content" json:"content"`
}

type mediaType struct {
	Schema *schemaObj `yaml:"schema" json:"schema"`
}

type response struct {
	Description string               `yaml:"description" json:"description"`
	Content     map[string]mediaType `yaml:"content" json:"content"`
}

type schemaObj struct {
	Type       string                `yaml:"type" json:"type"`
	Format     string                `yaml:"format" json:"format"`
	Properties map[string]*schemaObj `yaml:"properties" json:"properties"`
	Items      *schemaObj            `yaml:"items" json:"items"`
	Ref        string                `yaml:"$ref" json:"$ref"`
}

func parseSpec(data []byte) (openAPISpec, error) {
	if len(data) == 0 {
		return openAPISpec{}, fmt.Errorf("empty spec")
	}

	var spec openAPISpec

	// Try JSON first (faster), fall back to YAML.
	if err := json.Unmarshal(data, &spec); err != nil {
		if err2 := yaml.Unmarshal(data, &spec); err2 != nil {
			return openAPISpec{}, fmt.Errorf("parse spec: %w", err2)
		}
	}

	return spec, nil
}

// diffPaths compares the paths section of two OpenAPI specs.
func diffPaths(prev, curr map[string]map[string]operation) []domain.DiffResult {
	var results []domain.DiffResult

	prevPaths := sortedKeys(prev)
	currPaths := sortedKeys(curr)

	// Removed paths
	for _, path := range prevPaths {
		if _, exists := curr[path]; !exists {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindRemoved,
				Path:   path,
				Before: "path",
			})
		}
	}

	// Added paths
	for _, path := range currPaths {
		if _, exists := prev[path]; !exists {
			results = append(results, domain.DiffResult{
				Kind:  domain.ChangeKindAdded,
				Path:  path,
				After: "path",
			})
		}
	}

	// Paths that exist in both — compare operations
	for _, path := range prevPaths {
		currOps, exists := curr[path]
		if !exists {
			continue
		}
		prevOps := prev[path]
		results = append(results, diffOperations(path, prevOps, currOps)...)
	}

	return results
}

// diffOperations compares HTTP methods (operations) within a single path.
func diffOperations(path string, prev, curr map[string]operation) []domain.DiffResult {
	var results []domain.DiffResult

	prevMethods := sortedKeys(prev)
	currMethods := sortedKeys(curr)

	for _, method := range prevMethods {
		if _, exists := curr[method]; !exists {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindRemoved,
				Path:   fmt.Sprintf("%s.%s", path, method),
				Before: "operation",
			})
		}
	}

	for _, method := range currMethods {
		if _, exists := prev[method]; !exists {
			results = append(results, domain.DiffResult{
				Kind:  domain.ChangeKindAdded,
				Path:  fmt.Sprintf("%s.%s", path, method),
				After: "operation",
			})
		}
	}

	// Operations that exist in both — compare parameters and responses
	for _, method := range prevMethods {
		currOp, exists := curr[method]
		if !exists {
			continue
		}
		prevOp := prev[method]
		opPath := fmt.Sprintf("%s.%s", path, method)
		results = append(results, diffParameters(opPath, prevOp.Parameters, currOp.Parameters)...)
		results = append(results, diffResponses(opPath, prevOp.Responses, currOp.Responses)...)
		results = append(results, diffRequestBody(opPath, prevOp.RequestBody, currOp.RequestBody)...)
	}

	return results
}

// diffParameters compares the parameter lists of two operations.
func diffParameters(opPath string, prev, curr []parameter) []domain.DiffResult {
	var results []domain.DiffResult

	prevMap := make(map[string]parameter)
	for _, p := range prev {
		key := p.In + ":" + p.Name
		prevMap[key] = p
	}

	currMap := make(map[string]parameter)
	for _, p := range curr {
		key := p.In + ":" + p.Name
		currMap[key] = p
	}

	for key, pParam := range prevMap {
		cParam, exists := currMap[key]
		if !exists {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindRemoved,
				Path:   fmt.Sprintf("%s.parameters.%s", opPath, key),
				Before: "parameter",
			})
			continue
		}
		// Check required change
		if pParam.Required != cParam.Required {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindTypeChanged,
				Path:   fmt.Sprintf("%s.parameters.%s.required", opPath, key),
				Before: fmt.Sprintf("%v", pParam.Required),
				After:  fmt.Sprintf("%v", cParam.Required),
			})
		}
		// Check type change
		if pParam.Schema != nil && cParam.Schema != nil {
			results = append(results, diffSchema(
				fmt.Sprintf("%s.parameters.%s.schema", opPath, key),
				pParam.Schema, cParam.Schema,
			)...)
		}
	}

	for key := range currMap {
		if _, exists := prevMap[key]; !exists {
			results = append(results, domain.DiffResult{
				Kind:  domain.ChangeKindAdded,
				Path:  fmt.Sprintf("%s.parameters.%s", opPath, key),
				After: "parameter",
			})
		}
	}

	return results
}

// diffResponses compares the response codes of two operations.
func diffResponses(opPath string, prev, curr map[string]response) []domain.DiffResult {
	var results []domain.DiffResult

	for code := range prev {
		if _, exists := curr[code]; !exists {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindRemoved,
				Path:   fmt.Sprintf("%s.responses.%s", opPath, code),
				Before: "response",
			})
		}
	}

	for code := range curr {
		if _, exists := prev[code]; !exists {
			results = append(results, domain.DiffResult{
				Kind:  domain.ChangeKindAdded,
				Path:  fmt.Sprintf("%s.responses.%s", opPath, code),
				After: "response",
			})
		}
	}

	// Responses that exist in both — compare content schemas
	for code, prevResp := range prev {
		currResp, exists := curr[code]
		if !exists {
			continue
		}
		respPath := fmt.Sprintf("%s.responses.%s", opPath, code)
		results = append(results, diffContent(respPath, prevResp.Content, currResp.Content)...)
	}

	return results
}

// diffRequestBody compares the request body of two operations.
func diffRequestBody(opPath string, prev, curr *requestBody) []domain.DiffResult {
	var results []domain.DiffResult
	rbPath := opPath + ".requestBody"

	if prev == nil && curr == nil {
		return nil
	}
	if prev == nil && curr != nil {
		results = append(results, domain.DiffResult{
			Kind:  domain.ChangeKindAdded,
			Path:  rbPath,
			After: "requestBody",
		})
		return results
	}
	if prev != nil && curr == nil {
		results = append(results, domain.DiffResult{
			Kind:   domain.ChangeKindRemoved,
			Path:   rbPath,
			Before: "requestBody",
		})
		return results
	}

	if prev.Required != curr.Required {
		results = append(results, domain.DiffResult{
			Kind:   domain.ChangeKindTypeChanged,
			Path:   rbPath + ".required",
			Before: fmt.Sprintf("%v", prev.Required),
			After:  fmt.Sprintf("%v", curr.Required),
		})
	}

	results = append(results, diffContent(rbPath, prev.Content, curr.Content)...)
	return results
}

// diffContent compares media type entries (e.g. application/json) and their schemas.
func diffContent(basePath string, prev, curr map[string]mediaType) []domain.DiffResult {
	var results []domain.DiffResult

	for mt := range prev {
		if _, exists := curr[mt]; !exists {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindRemoved,
				Path:   fmt.Sprintf("%s.content.%s", basePath, mt),
				Before: "mediaType",
			})
		}
	}

	for mt := range curr {
		if _, exists := prev[mt]; !exists {
			results = append(results, domain.DiffResult{
				Kind:  domain.ChangeKindAdded,
				Path:  fmt.Sprintf("%s.content.%s", basePath, mt),
				After: "mediaType",
			})
		}
	}

	for mt, prevMT := range prev {
		currMT, exists := curr[mt]
		if !exists {
			continue
		}
		if prevMT.Schema != nil && currMT.Schema != nil {
			schemaPath := fmt.Sprintf("%s.content.%s.schema", basePath, mt)
			results = append(results, diffSchema(schemaPath, prevMT.Schema, currMT.Schema)...)
		}
	}

	return results
}

// diffSchema compares two inline schema objects (no $ref resolution).
func diffSchema(path string, prev, curr *schemaObj) []domain.DiffResult {
	var results []domain.DiffResult

	// Skip if either side is a $ref — we don't resolve references.
	if prev.Ref != "" || curr.Ref != "" {
		return nil
	}

	if prev.Type != curr.Type {
		results = append(results, domain.DiffResult{
			Kind:   domain.ChangeKindTypeChanged,
			Path:   path + ".type",
			Before: prev.Type,
			After:  curr.Type,
		})
	}

	if prev.Format != curr.Format {
		results = append(results, domain.DiffResult{
			Kind:   domain.ChangeKindTypeChanged,
			Path:   path + ".format",
			Before: prev.Format,
			After:  curr.Format,
		})
	}

	// Compare properties (object schemas)
	for propName, prevProp := range prev.Properties {
		currProp, exists := curr.Properties[propName]
		if !exists {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindRemoved,
				Path:   path + ".properties." + propName,
				Before: schemaTypeName(prevProp),
			})
			continue
		}
		results = append(results, diffSchema(path+".properties."+propName, prevProp, currProp)...)
	}
	for propName, currProp := range curr.Properties {
		if _, exists := prev.Properties[propName]; !exists {
			results = append(results, domain.DiffResult{
				Kind:  domain.ChangeKindAdded,
				Path:  path + ".properties." + propName,
				After: schemaTypeName(currProp),
			})
		}
	}

	// Compare array items
	if prev.Items != nil && curr.Items != nil {
		results = append(results, diffSchema(path+".items", prev.Items, curr.Items)...)
	}

	return results
}

func schemaTypeName(s *schemaObj) string {
	if s == nil {
		return "unknown"
	}
	if s.Ref != "" {
		return "$ref"
	}
	if s.Type != "" {
		return s.Type
	}
	return "object"
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
