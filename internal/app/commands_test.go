package app

import (
	"testing"

	tpl "pkg.gostartkit.com/dbx/internal/template"
)

func TestRedactTemplateValues(t *testing.T) {
	t.Parallel()

	template := &tpl.Template{
		Inputs: []tpl.Input{
			{Name: "password", Secret: true},
			{Name: "database", Secret: false},
		},
	}

	values := map[string]string{
		"database": "appdb",
		"password": "super-secret",
	}

	redacted := redactTemplateValues(template, values)

	if redacted["password"] != "***" {
		t.Fatalf("password redaction = %q, want ***", redacted["password"])
	}
	if redacted["database"] != "appdb" {
		t.Fatalf("database value = %q, want appdb", redacted["database"])
	}
	if values["password"] != "super-secret" {
		t.Fatalf("original values were mutated")
	}
}
