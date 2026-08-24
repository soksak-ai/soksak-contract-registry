package registry

import (
	"os"
	"strings"
	"testing"
)

func TestMakeOwnsRegistryContractCommands(t *testing.T) {
	body, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"preflight:", "prepare:", "build:", "verify:"} {
		if !strings.Contains(string(body), target) {
			t.Errorf("Makefile omits %s", target)
		}
	}
}
