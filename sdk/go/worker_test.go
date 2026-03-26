package magic_test

import (
	"testing"

	magic "github.com/kienbui1995/magic/sdk/go"
)

func TestWorkerCapability(t *testing.T) {
	w := magic.NewWorker("TestBot", "http://localhost:9000", 3)
	w.Capability("greet", "Says hello", 0.0, func(input map[string]any) (map[string]any, error) {
		name, _ := input["name"].(string)
		return map[string]any{"result": "Hello, " + name}, nil
	})
	result, err := w.HandleTask("greet", map[string]any{"name": "World"})
	if err != nil {
		t.Fatal(err)
	}
	if result["result"] != "Hello, World" {
		t.Fatalf("unexpected: %v", result)
	}
}
