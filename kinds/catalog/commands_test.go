package catalog

import "testing"

// A subcommand is named with its container, as code sends it.
func TestASubcommandIsNamedWithItsContainer(t *testing.T) {
	name, _, err := command([]byte(`{"KILL": {"summary": "x", "container": "CLIENT"}}`))
	if err != nil || name != "CLIENT KILL" {
		t.Errorf("%q, %v", name, err)
	}
	if _, _, err := command([]byte(`{"A": {}, "B": {}}`)); err == nil {
		t.Error("two commands in one file must be an error, not a guess")
	}
}
