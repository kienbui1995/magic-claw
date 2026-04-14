package llm

import (
	"context"
	"testing"
)

// mockProvider for testing
type mockProvider struct {
	name   string
	models []ModelInfo
	resp   *ChatResponse
	err    error
}

func (m *mockProvider) Name() string      { return m.name }
func (m *mockProvider) Models() []ModelInfo { return m.models }
func (m *mockProvider) Chat(_ context.Context, model string, _ []Message, _ int) (*ChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	r := *m.resp
	r.Model = model
	r.Provider = m.name
	return &r, nil
}

func TestGateway_RouteBest(t *testing.T) {
	gw := NewGateway()
	gw.RegisterProvider(&mockProvider{
		name:   "cheap",
		models: []ModelInfo{{ID: "cheap-1", Provider: "cheap", Quality: 50, Speed: 90, InputCostPer1K: 0.001}},
		resp:   &ChatResponse{Content: "cheap reply", Usage: Usage{TotalTokens: 10}},
	})
	gw.RegisterProvider(&mockProvider{
		name:   "good",
		models: []ModelInfo{{ID: "good-1", Provider: "good", Quality: 95, Speed: 60, InputCostPer1K: 0.01}},
		resp:   &ChatResponse{Content: "good reply", Usage: Usage{TotalTokens: 10}},
	})

	resp, err := gw.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		Strategy: "best",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Provider != "good" {
		t.Errorf("expected good provider, got %s", resp.Provider)
	}
}

func TestGateway_RouteCheapest(t *testing.T) {
	gw := NewGateway()
	gw.RegisterProvider(&mockProvider{
		name:   "expensive",
		models: []ModelInfo{{ID: "exp-1", Provider: "expensive", InputCostPer1K: 0.01, OutputCostPer1K: 0.03}},
		resp:   &ChatResponse{Content: "exp", Usage: Usage{TotalTokens: 10}},
	})
	gw.RegisterProvider(&mockProvider{
		name:   "cheap",
		models: []ModelInfo{{ID: "cheap-1", Provider: "cheap", InputCostPer1K: 0.0001, OutputCostPer1K: 0.0003}},
		resp:   &ChatResponse{Content: "cheap", Usage: Usage{TotalTokens: 10}},
	})

	resp, err := gw.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		Strategy: "cheapest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Provider != "cheap" {
		t.Errorf("expected cheap, got %s", resp.Provider)
	}
}

func TestGateway_Fallback(t *testing.T) {
	gw := NewGateway()
	gw.RegisterProvider(&mockProvider{
		name:   "broken",
		models: []ModelInfo{{ID: "b-1", Provider: "broken", Quality: 99}},
		err:    context.DeadlineExceeded,
	})
	gw.RegisterProvider(&mockProvider{
		name:   "backup",
		models: []ModelInfo{{ID: "bk-1", Provider: "backup", Quality: 50}},
		resp:   &ChatResponse{Content: "backup reply", Usage: Usage{TotalTokens: 5}},
	})

	resp, err := gw.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Provider != "backup" {
		t.Errorf("expected backup fallback, got %s", resp.Provider)
	}
}

func TestGateway_SpecificModel(t *testing.T) {
	gw := NewGateway()
	gw.RegisterProvider(&mockProvider{
		name:   "openai",
		models: []ModelInfo{{ID: "gpt-4o", Provider: "openai", Quality: 95}},
		resp:   &ChatResponse{Content: "gpt4 reply", Usage: Usage{TotalTokens: 20}},
	})

	resp, err := gw.Chat(context.Background(), ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", resp.Model)
	}
}

func TestEstimateTokens(t *testing.T) {
	tokens := EstimateTokens("Hello, world! This is a test.")
	if tokens < 5 || tokens > 10 {
		t.Errorf("expected ~7 tokens, got %d", tokens)
	}
}

func TestCalcCost(t *testing.T) {
	m := ModelInfo{InputCostPer1K: 0.01, OutputCostPer1K: 0.03}
	u := Usage{PromptTokens: 1000, CompletionTokens: 500}
	cost := calcCost(m, u)
	expected := 0.01 + 0.015 // 1K input * 0.01 + 0.5K output * 0.03
	if cost < expected-0.001 || cost > expected+0.001 {
		t.Errorf("expected %.4f, got %.4f", expected, cost)
	}
}
