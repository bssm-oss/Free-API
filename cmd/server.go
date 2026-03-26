package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/bssm-oss/Free-API/internal/config"
	appctx "github.com/bssm-oss/Free-API/internal/context"
	"github.com/bssm-oss/Free-API/internal/logging"
	"github.com/bssm-oss/Free-API/internal/models"
	"github.com/bssm-oss/Free-API/internal/provider"
	"github.com/spf13/cobra"
)

var serverAddr string

var swaggerHTML = template.Must(template.New("swagger").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>freeapi API Docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body {
      margin: 0;
      background: #fafafa;
    }
    .topbar {
      display: none;
    }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      displayRequestDuration: true,
      docExpansion: 'list',
      defaultModelsExpandDepth: 1
    });
  </script>
</body>
</html>`))

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
  GET  /               Swagger UI
  GET  /swagger        Swagger UI
  GET  /openapi.json   OpenAPI spec
  POST /freeapi/chat   JSON body: {"message":"hello"}
  GET  /healthz        Health check`,
	RunE: runServer,
}

func init() {
	serverCmd.Flags().StringVar(&serverAddr, "addr", "127.0.0.1:8080", "HTTP listen address")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err == nil {
		logging.Configure(cfg.LogPath, cfg.LogLevel)
	}
	srv := &http.Server{
		Addr:              serverAddr,
		Handler:           newServerMux(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Fprintf(os.Stderr, "freeapi server listening on http://%s\n", serverAddr)
	fmt.Fprintf(os.Stderr, "Docs: http://%s/swagger\n", serverAddr)
	fmt.Fprintf(os.Stderr, "POST http://%s/freeapi/chat\n", serverAddr)
	logging.Info("server.start", map[string]any{
		"addr": serverAddr,
	})

	return srv.ListenAndServe()
}

func newServerMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleServerRoot)
	mux.HandleFunc("/swagger", handleSwaggerUI)
	mux.HandleFunc("/openapi.json", handleOpenAPI)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/freeapi/chat", handleServerChat)
	return mux
}

func handleServerRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/swagger", http.StatusTemporaryRedirect)
}

func handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := swaggerHTML.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, openAPISpec())
}

func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "freeapi local server",
			"version":     Version,
			"description": "Local HTTP API for freeapi chat requests.",
		},
		"servers": []map[string]any{
			{"url": "/"},
		},
		"paths": map[string]any{
			"/healthz": map[string]any{
				"get": map[string]any{
					"summary":     "Health check",
					"operationId": "healthz",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "OK",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/HealthResponse",
									},
								},
							},
						},
					},
				},
			},
			"/freeapi/chat": map[string]any{
				"post": map[string]any{
					"summary":     "Send a chat request",
					"operationId": "chat",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"$ref": "#/components/schemas/ServerChatRequest",
								},
								"example": map[string]any{
									"message": "hello",
								},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Chat response",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/ServerChatResponse",
									},
								},
							},
						},
						"400": map[string]any{
							"description": "Invalid request",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
						"502": map[string]any{
							"description": "Provider error",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/ErrorResponse",
									},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"ServerChatRequest": map[string]any{
					"type":     "object",
					"required": []string{"message"},
					"properties": map[string]any{
						"message": map[string]any{
							"type":        "string",
							"description": "User message to send",
						},
						"cid": map[string]any{
							"type":        "string",
							"description": "Conversation ID or short ID prefix",
						},
						"continue": map[string]any{
							"type":        "boolean",
							"description": "Continue the last conversation when no ID is provided",
						},
						"conversation_id": map[string]any{
							"type":        "string",
							"description": "Exact conversation ID",
						},
						"provider": map[string]any{
							"type":        "string",
							"description": "Specific provider override",
						},
						"model": map[string]any{
							"type":        "string",
							"description": "Specific model override",
						},
						"system": map[string]any{
							"type":        "string",
							"description": "System prompt override",
						},
						"timeout": map[string]any{
							"type":        "integer",
							"description": "Request timeout in seconds",
							"default":     defaultChatTimeoutSeconds,
						},
					},
				},
				"ServerChatResponse": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"response":        map[string]any{"type": "string"},
						"content":         map[string]any{"type": "string"},
						"provider":        map[string]any{"type": "string"},
						"model":           map[string]any{"type": "string"},
						"conversation_id": map[string]any{"type": "string"},
						"tokens_in":       map[string]any{"type": "integer"},
						"tokens_out":      map[string]any{"type": "integer"},
						"elapsed_ms":      map[string]any{"type": "integer", "format": "int64"},
					},
				},
				"HealthResponse": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"status": map[string]any{"type": "string", "example": "ok"},
					},
				},
				"ErrorResponse": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"error": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleServerChat(w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("srv-%d", time.Now().UnixNano())
	start := time.Now()
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		logging.Error("server.request_rejected", map[string]any{
			"request_id":  requestID,
			"method":      r.Method,
			"path":        r.URL.Path,
			"http_status": http.StatusMethodNotAllowed,
			"error":       "method not allowed",
		})
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	defer r.Body.Close()

	var req serverChatRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		logging.Error("server.request_rejected", map[string]any{
			"request_id":  requestID,
			"http_status": http.StatusBadRequest,
			"error":       fmt.Sprintf("invalid JSON body: %v", err),
		})
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
		return
	}
	if req.Message == "" {
		logging.Error("server.request_rejected", map[string]any{
			"request_id":  requestID,
			"http_status": http.StatusBadRequest,
			"error":       "message is required",
		})
		writeJSONError(w, http.StatusBadRequest, "message is required")
		return
	}
	logging.Info("server.request_start", map[string]any{
		"request_id":        requestID,
		"method":            r.Method,
		"path":              r.URL.Path,
		"remote_addr":       r.RemoteAddr,
		"provider_override": req.Provider,
		"message_len":       len(req.Message),
		"timeout_s":         req.Timeout,
	})

	resp, err := executeServerChat(r.Context(), req)
	if err != nil {
		logging.Error("server.request_error", map[string]any{
			"request_id":  requestID,
			"elapsed_ms":  time.Since(start).Milliseconds(),
			"error":       err.Error(),
			"http_status": http.StatusBadGateway,
		})
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	logging.Info("server.request_finish", map[string]any{
		"request_id":      requestID,
		"elapsed_ms":      time.Since(start).Milliseconds(),
		"provider":        resp.Provider,
		"model":           resp.Model,
		"conversation_id": resp.ConversationID,
		"response_len":    len(resp.Content),
		"http_status":     http.StatusOK,
	})
	writeJSON(w, http.StatusOK, resp)
}

func executeServerChat(parent context.Context, req serverChatRequest) (*serverChatResponse, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	logging.Configure(cfg.LogPath, cfg.LogLevel)

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
