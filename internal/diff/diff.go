package diff

import (
	"fmt"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// Compare returns the list of structural changes between two snapshots.
func Compare(before, after types.Snapshot) []types.DiffResult {
	var results []types.DiffResult

	if before.StatusCode != after.StatusCode {
		results = append(results, types.DiffResult{
			Kind:   types.ChangeKindStatusChanged,
			Path:   "status",
			Before: fmt.Sprintf("%d", before.StatusCode),
			After:  fmt.Sprintf("%d", after.StatusCode),
		})
	}

	results = append(results, diffObjects(before.Body, after.Body, "")...)
	return results
}

func diffObjects(before, after map[string]any, prefix string) []types.DiffResult {
	var results []types.DiffResult

	for key, bVal := range before {
		path := joinPath(prefix, key)
		aVal, exists := after[key]
		if !exists {
			results = append(results, types.DiffResult{
				Kind:   types.ChangeKindRemoved,
				Path:   path,
				Before: typeName(bVal),
			})
			continue
		}
		results = append(results, compareValues(path, bVal, aVal)...)
	}

	for key, aVal := range after {
		path := joinPath(prefix, key)
		if _, exists := before[key]; !exists {
			results = append(results, types.DiffResult{
				Kind:  types.ChangeKindAdded,
				Path:  path,
				After: typeName(aVal),
			})
		}
	}

	return results
}

func compareValues(path string, before, after any) []types.DiffResult {
	// nullability change: one side is nil, the other is not (same structural type otherwise)
	if (before == nil) != (after == nil) {
		return []types.DiffResult{{
			Kind:   types.ChangeKindNullabilityChanged,
			Path:   path,
			Before: typeName(before),
			After:  typeName(after),
		}}
	}

	bName, aName := typeName(before), typeName(after)

	if bName != aName {
		return []types.DiffResult{{
			Kind:   types.ChangeKindTypeChanged,
			Path:   path,
			Before: bName,
			After:  aName,
		}}
	}

	// recurse into nested objects
	if bObj, ok := before.(map[string]any); ok {
		aObj := after.(map[string]any)
		return diffObjects(bObj, aObj, path)
	}

	// recurse into arrays
	if bArr, ok := before.([]any); ok {
		aArr := after.([]any)
		return diffArrays(path, bArr, aArr)
	}

	return nil
}

func diffArrays(path string, before, after []any) []types.DiffResult {
	bEmpty, aEmpty := len(before) == 0, len(after) == 0

	if bEmpty && aEmpty {
		return nil
	}

	if bEmpty != aEmpty {
		bDesc, aDesc := arrayDesc(before), arrayDesc(after)
		return []types.DiffResult{{
			Kind:   types.ChangeKindTypeChanged,
			Path:   path,
			Before: bDesc,
			After:  aDesc,
		}}
	}

	// both non-empty: compare element schemas using first element of each
	bElem, aElem := before[0], after[0]
	bName, aName := typeName(bElem), typeName(aElem)

	// element type changed
	if bName != aName {
		return []types.DiffResult{{
			Kind:   types.ChangeKindTypeChanged,
			Path:   path + "[]",
			Before: bName,
			After:  aName,
		}}
	}

	// both are arrays of objects: recurse into schema
	if bObj, ok := bElem.(map[string]any); ok {
		aObj := aElem.(map[string]any)
		return diffObjects(bObj, aObj, path+"[]")
	}

	return nil
}

func typeName(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

func arrayDesc(arr []any) string {
	if len(arr) == 0 {
		return "array(empty)"
	}
	return "array(" + typeName(arr[0]) + ")"
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
