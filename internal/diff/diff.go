package diff

import (
	"fmt"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

// Compare returns the list of structural changes between two snapshots.
func Compare(before, after domain.Snapshot) []domain.DiffResult {
	var results []domain.DiffResult

	if before.StatusCode != after.StatusCode {
		results = append(results, domain.DiffResult{
			Kind:   domain.ChangeKindStatusChanged,
			Path:   "status",
			Before: fmt.Sprintf("%d", before.StatusCode),
			After:  fmt.Sprintf("%d", after.StatusCode),
		})
	}

	results = append(results, diffObjects(before.Body, after.Body, "")...)
	return results
}

func diffObjects(before, after map[string]any, prefix string) []domain.DiffResult {
	var results []domain.DiffResult

	for key, bVal := range before {
		path := joinPath(prefix, key)
		aVal, exists := after[key]
		if !exists {
			results = append(results, domain.DiffResult{
				Kind:   domain.ChangeKindRemoved,
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
			results = append(results, domain.DiffResult{
				Kind:  domain.ChangeKindAdded,
				Path:  path,
				After: typeName(aVal),
			})
		}
	}

	return results
}

func compareValues(path string, before, after any) []domain.DiffResult {
	// nullability change: one side is nil, the other is not (same structural type otherwise)
	if (before == nil) != (after == nil) {
		return []domain.DiffResult{{
			Kind:   domain.ChangeKindNullabilityChanged,
			Path:   path,
			Before: typeName(before),
			After:  typeName(after),
		}}
	}

	bName, aName := typeName(before), typeName(after)

	if bName != aName {
		return []domain.DiffResult{{
			Kind:   domain.ChangeKindTypeChanged,
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

// diffArrays compares two arrays by inspecting only their first element as a
// representative of the array's schema. This means:
//   - Heterogeneous arrays (e.g. [1, "foo", {}]) are not fully analysed; only
//     the first element is used to infer the element type.
//   - Nested arrays (arrays of arrays) are not recursed into beyond the first
//     level; only the element type is compared.
//   - Changes in array length are not reported; only structural/type changes
//     in the element schema are detected.
func diffArrays(path string, before, after []any) []domain.DiffResult {
	bEmpty, aEmpty := len(before) == 0, len(after) == 0

	if bEmpty && aEmpty {
		return nil
	}

	if bEmpty != aEmpty {
		bDesc, aDesc := arrayDesc(before), arrayDesc(after)
		return []domain.DiffResult{{
			Kind:   domain.ChangeKindTypeChanged,
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
		return []domain.DiffResult{{
			Kind:   domain.ChangeKindTypeChanged,
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
