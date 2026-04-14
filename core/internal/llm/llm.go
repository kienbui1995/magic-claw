// Package llm provides a multi-provider LLM gateway with model routing,
// token counting, cost tracking, and automatic fallback.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ChatRequest is the input to the LLM gateway.
type ChatRequest struct {
	Model    string    `json:"model,omitempty"`    // specific model or empty for auto-route
	Messages []Message `json:"messages"`
	Strategy string    `json:"strategy,omitempty"` // cheapest, fastest, best (default)
	MaxTokens int     `json:"max_tokens,omitempty"`
}

// ChatResponse is the output from the LLM gateway.
type ChatResponse struct {
	ID       string  `json:"id"`
	Model    string  `json:"model"`
	Provider string  `json:"provider"`
	Content  string  `json:"content"`
	Usage    Usage   `json:"usage"`
	Cost     float64 `json:"cost"`
	Latency  int64   `json:"latency_ms"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider is the interface for LLM backends.
type Provider interface {
	Name() string
	Chat(ctx context.Context, model string, messages []Message, maxTokens int) (*ChatResponse, error)
	Models() []ModelInfo
}

// ModelInfo describes a model's capabilities and pricing.
type ModelInfo struct {
	ID              string  `json:"id"`
	Provider        string  `json:"provider"`
	InputCostPer1K  float64 `json:"input_cost_per_1k"`  // USD per 1K input tokens
	OutputCostPer1K float64 `json:"output_cost_per_1k"` // USD per 1K output tokens
	MaxContext       int    `json:"max_context"`
	Quality          int    `json:"quality"`  // 1-100 subjective quality score
	Speed            int    `json:"speed"`    // 1-100 speed score
}

// OnCost is called after each LLM request with cost info.
// Set this to integrate with external cost tracking (e.g., costctrl).
type OnCost func(model, provider string, cost float64, usage Usage)

// Gateway is the LLM routing gateway.
type Gateway struct {
	mu        sync.RWMutex
	providers map[string]Provider
	models    map[string]ModelInfo // model ID -> info
	history   []ChatResponse      // recent completions for cost tracking
	OnCost    OnCost              // optional callback
}

// NewGateway creates a new LLM gateway.
func NewGateway() *Gateway {
	return &Gateway{
		providers: make(map[string]Provider),
		models:    make(map[string]ModelInfo),
	}
}

const maxHistory = 10_000

// RegisterProvider adds an LLM provider to the gateway.
func (g *Gateway) RegisterProvider(p Provider) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.providers[p.Name()] = p
	for _, m := range p.Models() {
		g.models[m.ID] = m
	}
}

// Chat routes a request to the best provider/model based on strategy.
func (g *Gateway) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	g.mu.RLock()
	if len(g.providers) == 0 {
		g.mu.RUnlock()
		return nil, fmt.Errorf("no LLM providers registered")
	}
	model, provider, err := g.route(req)
	g.mu.RUnlock()
	if err != nil {
		return nil, err
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	start := time.Now()
	resp, err := provider.Chat(ctx, model.ID, req.Messages, maxTokens)
	if err != nil {
		// Fallback: try other providers
		g.mu.RLock()
		resp, err = g.fallback(ctx, model.ID, req.Messages, maxTokens, provider.Name())
		g.mu.RUnlock()
		if err != nil {
			return nil, err
		}
	}

	resp.Latency = time.Since(start).Milliseconds()
	resp.Cost = calcCost(model, resp.Usage)

	// Track history under write lock
	g.mu.Lock()
	g.history = append(g.history, *resp)
	if len(g.history) > maxHistory {
		g.history = g.history[len(g.history)-maxHistory:]
	}
	onCost := g.OnCost
	g.mu.Unlock()

	if onCost != nil {
		onCost(resp.Model, resp.Provider, resp.Cost, resp.Usage)
	}

	return resp, nil
}

func (g *Gateway) route(req ChatRequest) (ModelInfo, Provider, error) {
	// Specific model requested
	if req.Model != "" {
		m, ok := g.models[req.Model]
		if !ok {
			return ModelInfo{}, nil, fmt.Errorf("model %q not found", req.Model)
		}
		p, ok := g.providers[m.Provider]
		if !ok {
			return ModelInfo{}, nil, fmt.Errorf("provider %q not available", m.Provider)
		}
		return m, p, nil
	}

	// Auto-route by strategy
	strategy := req.Strategy
	if strategy == "" {
		strategy = "best"
	}

	var best ModelInfo
	found := false
	for _, m := range g.models {
		if !found {
			best = m
			found = true
			continue
		}
		switch strategy {
		case "cheapest":
			if m.InputCostPer1K+m.OutputCostPer1K < best.InputCostPer1K+best.OutputCostPer1K {
				best = m
			}
		case "fastest":
			if m.Speed > best.Speed {
				best = m
			}
		default: // "best"
			if m.Quality > best.Quality {
				best = m
			}
		}
	}
	if !found {
		return ModelInfo{}, nil, fmt.Errorf("no models available")
	}
	p := g.providers[best.Provider]
	return best, p, nil
}

func (g *Gateway) fallback(ctx context.Context, skipModel string, msgs []Message, maxTokens int, skipProvider string) (*ChatResponse, error) {
	for name, p := range g.providers {
		if name == skipProvider {
			continue
		}
		for _, m := range p.Models() {
			resp, err := p.Chat(ctx, m.ID, msgs, maxTokens)
			if err == nil {
				return resp, nil
			}
		}
	}
	return nil, fmt.Errorf("all providers failed")
}

func calcCost(m ModelInfo, u Usage) float64 {
	return (float64(u.PromptTokens)/1000)*m.InputCostPer1K +
		(float64(u.CompletionTokens)/1000)*m.OutputCostPer1K
}

// TotalCost returns total LLM spend.
func (g *Gateway) TotalCost() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var total float64
	for _, r := range g.history {
		total += r.Cost
	}
	return total
}

// ListModels returns all available models.
func (g *Gateway) ListModels() []ModelInfo {
	g.mu.RLock()
	defer g.mu.RUnlock()
	models := make([]ModelInfo, 0, len(g.models))
	for _, m := range g.models {
		models = append(models, m)
	}
	return models
}

// EstimateTokens gives a rough token count for text (1 token ≈ 4 chars for English).
func EstimateTokens(text string) int {
	return len(text) / 4
}

// --- OpenAI-compatible provider ---

// OpenAIProvider connects to OpenAI or any OpenAI-compatible API.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	models  []ModelInfo
}

// NewOpenAIProvider creates a provider for OpenAI API.
func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 120 * time.Second},
		models: []ModelInfo{
			{ID: "gpt-4o", Provider: "openai", InputCostPer1K: 0.0025, OutputCostPer1K: 0.01, MaxContext: 128000, Quality: 95, Speed: 70},
			{ID: "gpt-4o-mini", Provider: "openai", InputCostPer1K: 0.00015, OutputCostPer1K: 0.0006, MaxContext: 128000, Quality: 80, Speed: 90},
		},
	}
}

func (p *OpenAIProvider) Name() string      { return "openai" }
func (p *OpenAIProvider) Models() []ModelInfo { return p.models }

func (p *OpenAIProvider) Chat(ctx context.Context, model string, messages []Message, maxTokens int) (*ChatResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai: %d %s", resp.StatusCode, string(b))
	}

	var result struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &ChatResponse{
		ID:       result.ID,
		Model:    model,
		Provider: "openai",
		Content:  content,
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}, nil
}

// --- Anthropic provider ---

// AnthropicProvider connects to the Anthropic API.
type AnthropicProvider struct {
	apiKey string
	client *http.Client
	models []ModelInfo
}

// NewAnthropicProvider creates a provider for Anthropic API.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
		models: []ModelInfo{
			{ID: "claude-sonnet-4-20250514", Provider: "anthropic", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, MaxContext: 200000, Quality: 92, Speed: 75},
			{ID: "claude-haiku-4-20250514", Provider: "anthropic", InputCostPer1K: 0.0008, OutputCostPer1K: 0.004, MaxContext: 200000, Quality: 78, Speed: 95},
		},
	}
}

func (p *AnthropicProvider) Name() string      { return "anthropic" }
func (p *AnthropicProvider) Models() []ModelInfo { return p.models }

func (p *AnthropicProvider) Chat(ctx context.Context, model string, messages []Message, maxTokens int) (*ChatResponse, error) {
	// Extract system message
	var system string
	var msgs []Message
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		} else {
			msgs = append(msgs, m)
		}
	}

	payload := map[string]any{
		"model":      model,
		"messages":   msgs,
		"max_tokens": maxTokens,
	}
	if system != "" {
		payload["system"] = system
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic: %d %s", resp.StatusCode, string(b))
	}

	var result struct {
		ID      string `json:"id"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	content := ""
	if len(result.Content) > 0 {
		content = result.Content[0].Text
	}

	return &ChatResponse{
		ID:       result.ID,
		Model:    model,
		Provider: "anthropic",
		Content:  content,
		Usage: Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}

// --- Ollama provider (local models) ---

// OllamaProvider connects to a local Ollama instance.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
	models  []ModelInfo
}

// NewOllamaProvider creates a provider for local Ollama.
func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 300 * time.Second},
		models: []ModelInfo{
			{ID: "llama3", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, MaxContext: 8192, Quality: 65, Speed: 50},
		},
	}
}

func (p *OllamaProvider) Name() string      { return "ollama" }
func (p *OllamaProvider) Models() []ModelInfo { return p.models }

func (p *OllamaProvider) Chat(ctx context.Context, model string, messages []Message, maxTokens int) (*ChatResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &ChatResponse{
		ID:       fmt.Sprintf("ollama-%d", time.Now().UnixNano()),
		Model:    model,
		Provider: "ollama",
		Content:  result.Message.Content,
		Usage: Usage{
			PromptTokens:     result.PromptEvalCount,
			CompletionTokens: result.EvalCount,
			TotalTokens:      result.PromptEvalCount + result.EvalCount,
		},
	}, nil
}
