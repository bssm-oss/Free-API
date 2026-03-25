package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bssm-oss/Free-API/internal/config"
	appctx "github.com/bssm-oss/Free-API/internal/context"
	"github.com/bssm-oss/Free-API/internal/models"
	"github.com/bssm-oss/Free-API/internal/provider"
	"github.com/spf13/cobra"
)

var serverAddr string

type serverChatRequest struct {
	Message        string `json:"message"`
	CID            string `json:"cid,omitempty"`
	Continue       bool   `json:"continue,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	System         string `json:"system,omitempty"`
	Timeout        int    `json:"timeout,omitempty"`
}

type serverChatResponse struct {
	Response       string `json:"response"`
	Content        string `json:"content"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	ConversationID string `json:"conversation_id"`
	TokensIn       int    `json:"tokens_in,omitempty"`
	TokensOut      int    `json:"tokens_out,omitempty"`
	ElapsedMS      int64  `json:"elapsed_ms"`
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a local HTTP server",
	Long: `Start a local HTTP server for freeapi.

Endpoints:
  POST /freeapi/chat   JSON body: {"message":"hello"}
  GET  /healthz        Health check`,
	RunE: runServer,
}

func init() {
	serverCmd.Flags().StringVar(&serverAddr, "addr", "127.0.0.1:8080", "HTTP listen address")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	srv := &http.Server{
		Addr:              serverAddr,
		Handler:           newServerMux(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Fprintf(os.Stderr, "freeapi server listening on http://%s\n", serverAddr)
	fmt.Fprintf(os.Stderr, "POST http://%s/freeapi/chat\n", serverAddr)

	return srv.ListenAndServe()
}

func newServerMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/freeapi/chat", handleServerChat)
	return mux
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleServerChat(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	defer r.Body.Close()

	var req serverChatRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
		return
	}
	if req.Message == "" {
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}

	resp, err := executeServerChat(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func executeServerChat(parent context.Context, req serverChatRequest) (*serverChatResponse, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	dbPath := cfg.DBPath
	if dbPath == "" {
		dbPath = config.DefaultDBPath()
	}
	store, err := appctx.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	sysPrompt := cfg.DefaultSystemPrompt
	if req.System != "" {
		sysPrompt = req.System
	}
	mgr := appctx.NewManager(store, cfg.MaxContextMessages, cfg.ContextStrategy, sysPrompt)

	convID := req.ConversationID
	if convID == "" {
		convID = req.CID
	}

	convID, _, err = mgr.GetOrContinue(convID, req.Continue, sysPrompt)
	if err != nil {
		return nil, err
	}

	messages, err := mgr.BuildMessages(convID, req.Message)
	if err != nil {
		return nil, err
	}

	registry := provider.NewRegistry(cfg)
	rotator := provider.NewRotator(registry)

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 120
	}

	reqCtx, cancel := context.WithTimeout(parent, time.Duration(timeout)*time.Second)
	defer cancel()

	opts := models.ChatOptions{
		Model:  req.Model,
		Stream: false,
	}

	start := time.Now()
	var chatResp *models.Response
	if req.Provider != "" {
		chatResp, err = rotator.ChatWithProvider(reqCtx, req.Provider, messages, opts)
	} else {
		chatResp, err = rotator.Chat(reqCtx, messages, opts)
	}
	if err != nil {
		return nil, err
	}

	if err := mgr.SaveExchange(convID, req.Message, chatResp.Content, chatResp.Provider, chatResp.Model, chatResp.TokensIn, chatResp.TokensOut); err != nil {
		return nil, fmt.Errorf("save conversation: %w", err)
	}

	return &serverChatResponse{
		Response:       chatResp.Content,
		Content:        chatResp.Content,
		Provider:       chatResp.Provider,
		Model:          chatResp.Model,
		ConversationID: convID,
		TokensIn:       chatResp.TokensIn,
		TokensOut:      chatResp.TokensOut,
		ElapsedMS:      time.Since(start).Milliseconds(),
	}, nil
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
