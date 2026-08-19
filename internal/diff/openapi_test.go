package diff

import (
	"testing"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

func openAPISnap(raw string) domain.Snapshot {
	return domain.Snapshot{
		MonitorName: "test-spec",
		CapturedAt:  time.Now(),
		StatusCode:  200,
		RawBody:     []byte(raw),
	}
}

func TestOpenAPI_Diff_PathAdded(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
  /orders:
    get:
      responses:
        "200":
          description: OK
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindAdded, "/orders")
	if found == nil {
		t.Errorf("expected /orders to be reported as added, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_PathRemoved(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
  /orders:
    get:
      responses:
        "200":
          description: OK
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindRemoved, "/orders")
	if found == nil {
		t.Errorf("expected /orders to be reported as removed, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_OperationAdded(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
    post:
      responses:
        "201":
          description: Created
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindAdded, "/users.post")
	if found == nil {
		t.Errorf("expected /users.post to be reported as added, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_OperationRemoved(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
    delete:
      responses:
        "204":
          description: No Content
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindRemoved, "/users.delete")
	if found == nil {
		t.Errorf("expected /users.delete to be reported as removed, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_ParameterAdded(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      parameters:
        - name: page
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      parameters:
        - name: page
          in: query
          schema:
            type: integer
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindAdded, "/users.get.parameters.query:limit")
	if found == nil {
		t.Errorf("expected query:limit parameter to be added, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_ParameterRemoved(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      parameters:
        - name: page
          in: query
          schema:
            type: integer
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      parameters:
        - name: page
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindRemoved, "/users.get.parameters.query:limit")
	if found == nil {
		t.Errorf("expected query:limit parameter to be removed, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_ResponseAdded(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
        "404":
          description: Not Found
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindAdded, "/users.get.responses.404")
	if found == nil {
		t.Errorf("expected response 404 to be added, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_VersionChanged(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths: {}
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "2.0.0"
paths: {}
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindTypeChanged, "info.version")
	if found == nil {
		t.Errorf("expected info.version change, got: %v", diffs)
	}
	if found != nil && (found.Before != "1.0.0" || found.After != "2.0.0") {
		t.Errorf("expected 1.0.0 → 2.0.0, got %s → %s", found.Before, found.After)
	}
}

func TestOpenAPI_Diff_SchemaPropertyAdded(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
                  name:
                    type: string
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
                  name:
                    type: string
                  email:
                    type: string
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindAdded, "/users.get.responses.200.content.application/json.schema.properties.email")
	if found == nil {
		t.Errorf("expected email property to be added, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_SchemaPropertyTypeChanged(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindTypeChanged, "/users.get.responses.200.content.application/json.schema.properties.id.type")
	if found == nil {
		t.Errorf("expected id type change, got: %v", diffs)
	}
	if found != nil && (found.Before != "integer" || found.After != "string") {
		t.Errorf("expected integer → string, got %s → %s", found.Before, found.After)
	}
}

func TestOpenAPI_Diff_RefSkipped(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserV2"
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	// $ref schemas are skipped — no diffs should be reported for the schema content
	if len(diffs) != 0 {
		t.Errorf("expected no diffs for $ref schemas, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_NoChanges(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    get:
      parameters:
        - name: page
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
`
	prev := openAPISnap(spec)
	curr := openAPISnap(spec)

	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	if len(diffs) != 0 {
		t.Errorf("expected no changes, got: %v", diffs)
	}
}

func TestOpenAPI_Diff_JSONFormat(t *testing.T) {
	prev := openAPISnap(`{
  "openapi": "3.0.0",
  "info": {"version": "1.0.0"},
  "paths": {
    "/users": {
      "get": {
        "responses": {"200": {"description": "OK"}}
      }
    }
  }
}`)
	curr := openAPISnap(`{
  "openapi": "3.0.0",
  "info": {"version": "1.0.0"},
  "paths": {
    "/users": {
      "get": {
        "responses": {"200": {"description": "OK"}}
      }
    },
    "/orders": {
      "get": {
        "responses": {"200": {"description": "OK"}}
      }
    }
  }
}`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindAdded, "/orders")
	if found == nil {
		t.Errorf("expected /orders added (JSON format), got: %v", diffs)
	}
}

func TestOpenAPI_Diff_RequestBodyAdded(t *testing.T) {
	prev := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    post:
      responses:
        "201":
          description: Created
`)
	curr := openAPISnap(`
openapi: "3.0.0"
info:
  version: "1.0.0"
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
      responses:
        "201":
          description: Created
`)
	s := OpenAPI{}
	diffs := s.Diff(prev, curr)

	found := findDiff(diffs, domain.ChangeKindAdded, "/users.post.requestBody")
	if found == nil {
		t.Errorf("expected requestBody to be added, got: %v", diffs)
	}
}

// --- helpers ---

func findDiff(diffs []domain.DiffResult, kind domain.ChangeKind, path string) *domain.DiffResult {
	for i := range diffs {
		if diffs[i].Kind == kind && diffs[i].Path == path {
			return &diffs[i]
		}
	}
	return nil
}
