package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/wisdom/pkg/cortex"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	connStr := strings.TrimSpace(os.Getenv("DB_CONN_STRING"))
	if connStr == "" {
		log.Fatal("DB_CONN_STRING environment variable is required")
	}

	engine, err := cortex.NewPostgresEngine(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer engine.Close()

	mux := http.NewServeMux()

	// CORS Middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/whoami", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")

		// Validate with Google
		resp, err := http.Get("https://www.googleapis.com/oauth2/v3/userinfo?access_token=" + token)
		if err != nil || resp.StatusCode != 200 {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		defer resp.Body.Close()
		var userInfo map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	})

	mux.HandleFunc("/cortex/nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes, err := engine.GetAllNodes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Flatten payload for frontend
		type FlatNode struct {
			ID            string   `json:"id"`
			Type          string   `json:"type"`
			Confidence    float64  `json:"confidence"`
			RequiresHuman bool     `json:"requires_human"`
			Content       string   `json:"content"`
			Author        string   `json:"author"`
			Stratum       string   `json:"stratum"`
			EntityClass   string   `json:"entity_class"`
			ImpactScore   float64  `json:"impact_score"`
			CreatedAt     string   `json:"created_at"`
		}

		flatNodes := make([]FlatNode, 0, len(nodes))
		for _, n := range nodes {
			fn := FlatNode{
				ID:            n.ID,
				Type:          string(n.Type),
				Confidence:    n.Confidence,
				RequiresHuman: n.RequiresHuman,
				CreatedAt:     n.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
			if n.Payload != nil {
				if c, ok := n.Payload["content"].(string); ok {
					fn.Content = c
				}
				if a, ok := n.Payload["author"].(string); ok {
					fn.Author = a
				}
				if s, ok := n.Payload["stratum"].(string); ok {
					fn.Stratum = s
				}
				if ec, ok := n.Payload["entity_class"].(string); ok {
					fn.EntityClass = ec
				}
				if is, ok := n.Payload["impact_score"].(float64); ok {
					fn.ImpactScore = is
				}
			}
			flatNodes = append(flatNodes, fn)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flatNodes)
	})

	mux.HandleFunc("/cortex/edges", func(w http.ResponseWriter, r *http.Request) {
		edges, err := engine.GetAllEdges(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(edges)
	})

	// Static file serving for the Portal
	fs := http.FileServer(http.Dir("./public"))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If it's an API route that wasn't matched, it will fall through here
		// We should only serve static files for non-API routes or if the file exists
		if strings.HasPrefix(r.URL.Path, "/cortex") || strings.HasPrefix(r.URL.Path, "/whoami") || strings.HasPrefix(r.URL.Path, "/health") {
			http.NotFound(w, r)
			return
		}

		// Check if file exists in public
		path := "./public" + r.URL.Path
		_, err := os.Stat(path)
		if os.IsNotExist(err) || r.URL.Path == "/" {
			// Serve index.html for SPA routing
			http.ServeFile(w, r, "./public/index.html")
			return
		}

		fs.ServeHTTP(w, r)
	}))

	log.Printf("Wisdom REST API listening on port %s", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
