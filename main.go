// muninn is the page that watches the bots: who uses each one, how much traffic reaches it, and
// whether someone is trying to flood it. Bots opt in with a label; muninn only reads.
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

// buildVersion is set at build time to the release tag.
var buildVersion = "local"

//go:embed web
var webFS embed.FS

func main() {
	w := newWatcher(getenv("DOCKER_HOST_URL", "http://docker-socket-proxy:2375"))
	name := getenv("MUNINN_NAME", "")

	web, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(web)))
	mux.HandleFunc("/healthz", func(rw http.ResponseWriter, r *http.Request) { fmt.Fprintln(rw, "ok") })
	mux.HandleFunc("/api/avatar/telegram/", newAvatars().serve)
	mux.HandleFunc("/api/bots", func(rw http.ResponseWriter, r *http.Request) {
		bots, err := w.bots()
		if err != nil {
			writeJSON(rw, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(rw, http.StatusOK, map[string]any{"name": name, "version": buildVersion, "bots": bots})
	})
	mux.HandleFunc("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		bots, err := w.bots()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadGateway)
			return
		}
		rw.Header().Set("Content-Type", "text/plain; version=0.0.4")
		writeMetrics(rw, bots)
	})

	addr := getenv("LISTEN_ADDR", ":8095")
	log.Printf("muninn %s on %s, reading containers from %s", buildVersion, addr, w.docker)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(rw http.ResponseWriter, status int, v any) {
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("Cache-Control", "no-store")
	rw.WriteHeader(status)
	_ = json.NewEncoder(rw).Encode(v)
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
