package diff

import (
	"testing"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

func snap(status int, body map[string]any) types.Snapshot {
	return types.Snapshot{StatusCode: status, Body: body}
}

func TestCompare_NoChanges(t *testing.T) {
	body := map[string]any{"name": "foo", "age": float64(30)}
	results := Compare(snap(200, body), snap(200, body))
	if len(results) != 0 {
		t.Fatalf("expected no changes, got %d", len(results))
	}
}

func TestCompare_StatusChanged(t *testing.T) {
	results := Compare(snap(200, nil), snap(404, nil))
	if len(results) != 1 || results[0].Kind != types.ChangeKindStatusChanged {
		t.Fatalf("expected status change, got %v", results)
	}
	if results[0].Before != "200" || results[0].After != "404" {
		t.Fatalf("unexpected before/after: %v", results[0])
	}
}

func TestCompare_FieldAdded(t *testing.T) {
	before := snap(200, map[string]any{"name": "foo"})
	after := snap(200, map[string]any{"name": "foo", "email": "x@x.com"})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindAdded || results[0].Path != "email" {
		t.Fatalf("expected email added, got %v", results)
	}
}

func TestCompare_FieldRemoved(t *testing.T) {
	before := snap(200, map[string]any{"name": "foo", "email": "x@x.com"})
	after := snap(200, map[string]any{"name": "foo"})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindRemoved || results[0].Path != "email" {
		t.Fatalf("expected email removed, got %v", results)
	}
}

func TestCompare_TypeChanged(t *testing.T) {
	before := snap(200, map[string]any{"amount": "100"})
	after := snap(200, map[string]any{"amount": float64(100)})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindTypeChanged {
		t.Fatalf("expected type change, got %v", results)
	}
	if results[0].Before != "string" || results[0].After != "number" {
		t.Fatalf("unexpected types: %v", results[0])
	}
}

func TestCompare_NullabilityChanged(t *testing.T) {
	before := snap(200, map[string]any{"field": nil})
	after := snap(200, map[string]any{"field": "value"})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindNullabilityChanged {
		t.Fatalf("expected nullability change, got %v", results)
	}
}

func TestCompare_NestedFieldChanged(t *testing.T) {
	before := snap(200, map[string]any{"customer": map[string]any{"name": "foo", "doc": "123"}})
	after := snap(200, map[string]any{"customer": map[string]any{"name": "foo"}})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindRemoved || results[0].Path != "customer.doc" {
		t.Fatalf("expected customer.doc removed, got %v", results)
	}
}

func TestCompare_ArrayOfObjects_FieldRemoved(t *testing.T) {
	before := snap(200, map[string]any{
		"items": []any{map[string]any{"id": float64(1), "name": "foo"}},
	})
	after := snap(200, map[string]any{
		"items": []any{map[string]any{"id": float64(1)}},
	})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindRemoved || results[0].Path != "items[].name" {
		t.Fatalf("expected items[].name removed, got %v", results)
	}
}

func TestCompare_ArrayOfObjects_FieldAdded(t *testing.T) {
	before := snap(200, map[string]any{
		"items": []any{map[string]any{"id": float64(1)}},
	})
	after := snap(200, map[string]any{
		"items": []any{map[string]any{"id": float64(1), "price": float64(99)}},
	})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindAdded || results[0].Path != "items[].price" {
		t.Fatalf("expected items[].price added, got %v", results)
	}
}

func TestCompare_ArrayBecameEmpty(t *testing.T) {
	before := snap(200, map[string]any{
		"items": []any{map[string]any{"id": float64(1)}},
	})
	after := snap(200, map[string]any{
		"items": []any{},
	})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindTypeChanged || results[0].Path != "items" {
		t.Fatalf("expected items type change (empty), got %v", results)
	}
	if results[0].Before != "array(object)" || results[0].After != "array(empty)" {
		t.Fatalf("unexpected before/after: %v", results[0])
	}
}

func TestCompare_ArrayElementTypeChanged(t *testing.T) {
	before := snap(200, map[string]any{"tags": []any{"foo", "bar"}})
	after := snap(200, map[string]any{"tags": []any{float64(1), float64(2)}})
	results := Compare(before, after)
	if len(results) != 1 || results[0].Kind != types.ChangeKindTypeChanged || results[0].Path != "tags[]" {
		t.Fatalf("expected tags[] type change, got %v", results)
	}
	if results[0].Before != "string" || results[0].After != "number" {
		t.Fatalf("unexpected types: %v", results[0])
	}
}
