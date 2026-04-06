package entities

import "testing"

func TestMetadataUnmarshalCurrentFields(t *testing.T) {
	payload := []byte(`{
		"target":"compose",
		"phase":"post",
		"subject":"service",
		"metadata":{
			"id":"CP001",
			"name":"name",
			"description":"desc",
			"severity":"warn",
			"mitigation":"fix",
			"reference":"docs"
		},
		"expression":{"expr_kind":"field","select":"service","expr":{"op":"exists"}}
	}`)

	var rule BaseRule
	if err := rule.UnmarshalJSON(payload); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if rule.Metadata == nil {
		t.Fatalf("Metadata is nil")
	}
	if rule.Metadata.Mitigation != "fix" || rule.Metadata.Reference != "docs" || rule.Metadata.Description != "desc" {
		t.Fatalf("unexpected metadata: %+v", rule.Metadata)
	}
}
