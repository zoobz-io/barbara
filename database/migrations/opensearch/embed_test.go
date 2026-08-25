package opensearch

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/zoobz-io/barbara/database/models"
)

// TestDocumentsMappingLoads asserts the embedded documents mapping is valid
// JSON and that its property types match the plan: identifier-like fields are
// keyword (an analyzed key would break exact get-by-key), content is analyzed
// text.
func TestDocumentsMappingLoads(t *testing.T) {
	raw, err := Mappings.ReadFile("001_documents.json")
	if err != nil {
		t.Fatalf("reading embedded mapping: %v", err)
	}

	var doc struct {
		Mappings struct {
			Properties map[string]struct {
				Type string `json:"type"`
			} `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("mapping is not valid JSON: %v", err)
	}

	wantType := map[string]string{
		"document_id":    "keyword",
		"tenant_id":      "keyword",
		"version_id":     "keyword",
		"key":            "keyword",
		"tags":           "keyword",
		"version_number": "integer",
		"content":        "text",
		"created_at":     "date",
		"updated_at":     "date",
	}
	for field, typ := range wantType {
		got, ok := doc.Mappings.Properties[field]
		if !ok {
			t.Errorf("mapping missing property %q", field)
			continue
		}
		if got.Type != typ {
			t.Errorf("property %q: type = %q, want %q", field, got.Type, typ)
		}
	}
}

// TestMappingMatchesModel guards against struct/mapping drift: every JSON field
// on models.DocumentIndex must have a corresponding property in the mapping.
// Extraction will add fields later via a mapping change — this keeps the two in
// lockstep.
func TestMappingMatchesModel(t *testing.T) {
	raw, err := Mappings.ReadFile("001_documents.json")
	if err != nil {
		t.Fatalf("reading embedded mapping: %v", err)
	}
	var doc struct {
		Mappings struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("mapping is not valid JSON: %v", err)
	}

	typ := reflect.TypeOf(models.DocumentIndex{})
	for i := range typ.NumField() {
		tag := typ.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		if _, ok := doc.Mappings.Properties[name]; !ok {
			t.Errorf("DocumentIndex field %q (json:%q) has no mapping property", typ.Field(i).Name, name)
		}
	}
}
