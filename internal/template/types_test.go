package template

import (
	"encoding/json"
	"testing"
)

func TestInputEffectiveTypeAndDefaults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   Input
		want string
	}{
		{name: "legacy secret", in: Input{Secret: true}, want: "secret"},
		{name: "legacy identifier", in: Input{Identifier: true}, want: "identifier"},
		{name: "legacy choices", in: Input{Choices: []string{"a"}}, want: "select"},
		{name: "typed confirm", in: Input{Type: "confirm"}, want: "confirm"},
		{name: "default string", in: Input{}, want: "string"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.in.EffectiveType(); got != tc.want {
				t.Fatalf("EffectiveType() = %q, want %q", got, tc.want)
			}
		})
	}

	confirmInput := Input{Type: "confirm", Default: true}
	if !confirmInput.DefaultBool() {
		t.Fatalf("DefaultBool() = false, want true")
	}

	intInput := Input{Type: "int", Default: 3306}
	if got, ok := intInput.DefaultInt(); !ok || got != 3306 {
		t.Fatalf("DefaultInt() = %d, %t", got, ok)
	}

	selectInput := Input{Type: "select", Options: []string{"utf8mb4", "utf8"}, Default: "utf8mb4"}
	if got := selectInput.DefaultString(); got != "utf8mb4" {
		t.Fatalf("DefaultString() = %q", got)
	}
	if len(selectInput.SelectOptions()) != 2 {
		t.Fatalf("SelectOptions() length = %d", len(selectInput.SelectOptions()))
	}

	requiredInput := Input{Name: "database"}
	if !requiredInput.IsRequired() {
		t.Fatalf("IsRequired() = false, want true when required is omitted")
	}

	optionalByDefault := Input{Name: "charset", Default: "utf8mb4"}
	if optionalByDefault.IsRequired() {
		t.Fatalf("IsRequired() = true, want false when default exists")
	}

	optionalFlag := false
	explicitOptional := Input{Name: "password", Required: &optionalFlag}
	if explicitOptional.IsRequired() {
		t.Fatalf("IsRequired() = true, want false when required=false")
	}
}

func TestTemplateTransactionJSON(t *testing.T) {
	t.Parallel()

	var tmpl Template
	data := []byte(`{
		"name": "create_database_with_user",
		"transaction": true,
		"match": {"command": "create database", "driver": "mysql"},
		"actions": [{"type": "sql", "description": "Create database", "sql": "CREATE DATABASE demo"}]
	}`)

	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if !tmpl.Transaction {
		t.Fatalf("Transaction = false, want true")
	}
	if tmpl.Version != CurrentTemplateSchemaVersion {
		t.Fatalf("Version = %d", tmpl.Version)
	}
}

func TestTemplateCategoryDefaultAndTagsNormalization(t *testing.T) {
	t.Parallel()

	var tmpl Template
	data := []byte(`{
		"name": "readonly_user",
		"tags": ["Readonly", "grant", "readonly", ""],
		"match": {"command": "create user", "driver": "mysql"},
		"actions": [{"type": "sql", "description": "Create user", "sql": "CREATE USER ro"}]
	}`)

	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if tmpl.Category != DefaultTemplateCategory {
		t.Fatalf("Category = %q, want %q", tmpl.Category, DefaultTemplateCategory)
	}
	if len(tmpl.Tags) != 2 || tmpl.Tags[0] != "grant" || tmpl.Tags[1] != "readonly" {
		t.Fatalf("Tags = %#v, want normalized sorted tags", tmpl.Tags)
	}
}

func TestTemplateValidateRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	tmpl := &Template{
		Version: 2,
		Name:    "bad",
		Match: Match{
			Command: "create database",
		},
		Actions: []Action{{Type: "sql", Description: "x", SQL: "SELECT 1"}},
	}
	err := tmpl.Validate()
	if err == nil || err.Error() != "unsupported version 2" {
		t.Fatalf("Validate error = %v", err)
	}
}
