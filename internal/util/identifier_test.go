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
