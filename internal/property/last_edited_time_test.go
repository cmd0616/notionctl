package property

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestLastEditedTime(t *testing.T) {
	p, err := Get("last_edited_time")
	if err != nil {
		t.Fatal(err)
	}
	if p.Type() != "last_edited_time" {
		t.Fatalf("unexpected type: %q", p.Type())
	}
	for _, config := range []map[string]interface{}{nil, {}, {"value": "ignored"}} {
		cfg, err := p.ToNotion(config)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if string(payload) != `{"last_edited_time":{}}` {
			t.Errorf("unexpected payload: %s", payload)
		}
		if !reflect.DeepEqual(cfg, NotionPropertyConfig{"last_edited_time": map[string]interface{}{}}) {
			t.Errorf("unexpected config: %#v", cfg)
		}
		if summary := p.DiffSummary(config, map[string]interface{}{"value": "server-managed"}); summary != "" {
			t.Errorf("read-only property has configurable diff: %q", summary)
		}
	}
}
