package util

import "testing"

func TestValidateIdentifier(t *testing.T) {
	t.Parallel()

	valid := []string{"db1", "db_name", "_internal"}
	for _, name := range valid {
		if err := ValidateIdentifier(name); err != nil {
			t.Fatalf("ValidateIdentifier(%q) returned error: %v", name, err)
		}
	}

	invalid := []string{"", "1db", "db-name", "db name", "drop database"}
	for _, name := range invalid {
		if err := ValidateIdentifier(name); err == nil {
			t.Fatalf("ValidateIdentifier(%q) succeeded, want error", name)
		}
	}
}

func TestValidateDatabaseName(t *testing.T) {
	t.Parallel()

	valid := []string{"greenhn-dev", "prod-db", "analytics_v2", "app_v2", "db1"}
	for _, name := range valid {
		if err := ValidateDatabaseName(name); err != nil {
			t.Fatalf("ValidateDatabaseName(%q) returned error: %v", name, err)
		}
	}

	invalid := []string{"", "foo bar", "foo;drop", "foo`", "foo/db"}
	for _, name := range invalid {
		err := ValidateDatabaseName(name)
		if err == nil {
			t.Fatalf("ValidateDatabaseName(%q) succeeded, want error", name)
		}
		if name != "" && err.Error() != `invalid database name "`+name+`"` {
			t.Fatalf("ValidateDatabaseName(%q) error = %q", name, err.Error())
		}
	}
}
