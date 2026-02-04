package types

import (
	"encoding/json"
	"testing"
)

func TestGraph_JSONRoundTrip(t *testing.T) {
	orig := &Graph{
		ID:       "test-graph",
		Created:  "2025-01-01T00:00:00Z",
		Elements: []Element{
			{
				Group: "nodes",
				Data: ElementData{
					ID:    "github:org/repo/overlay@main",
					Label: "overlay",
					Type:  "overlay",
					Path:  "overlay",
				},
			},
			{
				Group: "nodes",
				Data: ElementData{
					ID:    "github:org/repo/base@main",
					Label: "base",
					Type:  "resource",
					Path:  "base",
				},
			},
			{
				Group: "edges",
				Data: ElementData{
					ID:       "overlay->base",
					Source:   "github:org/repo/overlay@main",
					Target:   "github:org/repo/base@main",
					EdgeType: "resource",
				},
			},
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Graph
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ID != orig.ID {
		t.Errorf("ID = %q, want %q", got.ID, orig.ID)
	}
	if got.Created != orig.Created {
		t.Errorf("Created = %q, want %q", got.Created, orig.Created)
	}
	if len(got.Elements) != len(orig.Elements) {
		t.Fatalf("Elements len = %d, want %d", len(got.Elements), len(orig.Elements))
	}
	for i := range orig.Elements {
		if got.Elements[i].Group != orig.Elements[i].Group {
			t.Errorf("Elements[%d].Group = %q, want %q", i, got.Elements[i].Group, orig.Elements[i].Group)
		}
		if got.Elements[i].Data.ID != orig.Elements[i].Data.ID {
			t.Errorf("Elements[%d].Data.ID = %q, want %q", i, got.Elements[i].Data.ID, orig.Elements[i].Data.ID)
		}
	}
}

func TestGraph_JSONShape(t *testing.T) {
	g := &Graph{
		ID:       "shape-test",
		Created:  "2025-01-01",
		Elements: []Element{},
	}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}
	for _, key := range []string{"id", "elements", "created"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("Graph JSON missing key %q (API/frontend contract)", key)
		}
	}
	if _, ok := raw["elements"]; ok {
		elems, _ := raw["elements"].([]interface{})
		if elems == nil {
			t.Errorf("elements should be present (array)")
		}
	}
}

func TestGraph_Empty(t *testing.T) {
	g := &Graph{}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("Marshal empty Graph: %v", err)
	}
	var got Graph
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal empty Graph: %v", err)
	}
	if got.Elements != nil && len(got.Elements) != 0 {
		t.Errorf("empty Graph Elements should be nil or empty, got len %d", len(got.Elements))
	}
}

func TestElement_NodeAndEdge(t *testing.T) {
	node := Element{
		Group: "nodes",
		Data: ElementData{
			ID:     "node-1",
			Label:  "overlay",
			Type:   "overlay",
			Path:   "overlay",
			Source: "", Target: "", EdgeType: "",
		},
	}
	edge := Element{
		Group: "edges",
		Data: ElementData{
			ID: "a->b", Source: "a", Target: "b", EdgeType: "resource",
			Label: "", Type: "", Path: "",
		},
	}

	for name, e := range map[string]Element{"node": node, "edge": edge} {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("Marshal %s: %v", name, err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal %s: %v", name, err)
		}
		if _, ok := raw["group"]; !ok {
			t.Errorf("%s: missing group", name)
		}
		if _, ok := raw["data"]; !ok {
			t.Errorf("%s: missing data", name)
		}
	}
}

func TestNodeDetails_JSONRoundTrip(t *testing.T) {
	orig := &NodeDetails{
		ID:       "node-1",
		Label:    "base",
		Type:     "resource",
		Path:     "components/base",
		Content:  map[string]interface{}{"resources": []interface{}{"deploy.yaml"}},
		Parents:  []string{"overlay-1"},
		Children: []string{"child-1", "child-2"},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal NodeDetails: %v", err)
	}

	var got NodeDetails
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal NodeDetails: %v", err)
	}

	if got.ID != orig.ID || got.Label != orig.Label || got.Type != orig.Type || got.Path != orig.Path {
		t.Errorf("NodeDetails fields: got ID=%q Label=%q Type=%q Path=%q", got.ID, got.Label, got.Type, got.Path)
	}
	if len(got.Parents) != len(orig.Parents) || len(got.Children) != len(orig.Children) {
		t.Errorf("Parents len=%d Children len=%d, want %d %d", len(got.Parents), len(got.Children), len(orig.Parents), len(orig.Children))
	}
}

func TestNodeDetails_JSONShape(t *testing.T) {
	d := &NodeDetails{
		ID: "id", Label: "l", Type: "overlay", Path: "p",
		Parents: []string{}, Children: []string{},
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, key := range []string{"id", "label", "type", "path", "parents", "children"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("NodeDetails JSON missing key %q (details endpoint contract)", key)
		}
	}
}
