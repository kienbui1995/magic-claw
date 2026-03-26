package magic

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type HandlerFunc func(input map[string]any) (map[string]any, error)

type capability struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	EstCostPerCall float64 `json:"est_cost_per_call"`
}

type Worker struct {
	name         string
	endpoint     string
	maxWorkers   int
	capabilities map[string]capability
	handlers     map[string]HandlerFunc
	workerID     string
	client       *Client
	sem          chan struct{}
	mu           sync.Mutex
}

func NewWorker(name, endpoint string, maxWorkers int) *Worker {
	return &Worker{
		name:         name,
		endpoint:     endpoint,
		maxWorkers:   maxWorkers,
		capabilities: make(map[string]capability),
		handlers:     make(map[string]HandlerFunc),
		sem:          make(chan struct{}, maxWorkers),
	}
}

func (w *Worker) Capability(name, description string, estCost float64, fn HandlerFunc) {
	w.capabilities[name] = capability{Name: name, Description: description, EstCostPerCall: estCost}
	w.handlers[name] = fn
}

func (w *Worker) HandleTask(taskType string, input map[string]any) (map[string]any, error) {
	fn, ok := w.handlers[taskType]
	if !ok {
		return nil, fmt.Errorf("no handler for %s", taskType)
	}
	return fn(input)
}

func (w *Worker) Register(magicURL, apiKey string) error {
	w.client = NewClient(magicURL, apiKey)
	caps := make([]capability, 0, len(w.capabilities))
	for _, c := range w.capabilities {
		caps = append(caps, c)
	}
	payload := map[string]any{
		"name":         w.name,
		"capabilities": caps,
		"endpoint":     map[string]string{"type": "http", "url": w.endpoint},
		"limits":       map[string]any{"max_concurrent_tasks": w.maxWorkers},
	}
	result, err := w.client.RegisterWorker(payload)
	if err != nil {
		return err
	}
	if id, ok := result["id"].(string); ok {
		w.workerID = id
	}
	slog.Info("registered", "worker_id", w.workerID)
	return nil
}

func (w *Worker) startHeartbeat() {
	go func() {
		for range time.Tick(30 * time.Second) {
			if w.client != nil && w.workerID != "" {
				if err := w.client.Heartbeat(w.workerID); err != nil {
					slog.Warn("heartbeat failed", "err", err)
				}
			}
		}
	}()
}

func (w *Worker) Serve(addr string) error {
	w.startHeartbeat()
	mux := http.NewServeMux()
	mux.HandleFunc("/", w.handleHTTP)
	slog.Info("serving", "name", w.name, "addr", addr)
	return http.ListenAndServe(addr, mux)
}

func (w *Worker) handleHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(rw, "bad json", http.StatusBadRequest)
		return
	}
	msgType, _ := body["type"].(string)
	payload, _ := body["payload"].(map[string]any)
	if msgType != "task.assign" {
		http.Error(rw, "unknown type", http.StatusNotFound)
		return
	}

	taskID, _ := payload["task_id"].(string)
	taskType, _ := payload["task_type"].(string)
	input, _ := payload["input"].(map[string]any)

	// acquire semaphore (non-blocking)
	select {
	case w.sem <- struct{}{}:
	default:
		respond(rw, map[string]any{
			"type":    "task.fail",
			"payload": map[string]any{"task_id": taskID, "error": map[string]string{"code": "overloaded", "message": "at max capacity"}},
		})
		return
	}
	defer func() { <-w.sem }()

	result, err := w.HandleTask(taskType, input)
	if err != nil {
		respond(rw, map[string]any{
			"type":    "task.fail",
			"payload": map[string]any{"task_id": taskID, "error": map[string]string{"code": "handler_error", "message": err.Error()}},
		})
		return
	}
	respond(rw, map[string]any{
		"type":    "task.complete",
		"payload": map[string]any{"task_id": taskID, "output": result, "cost": 0.0},
	})
}

func respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
