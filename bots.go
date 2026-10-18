package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PlatanosVerdes/gonana/botstats"
)

// statsLabel is the one label a bot's container carries to show up here: the URL of its
// /bot-stats document, on a port muninn reaches over the Docker network.
const statsLabel = "muninn.stats"

// bot is one container that asked to be watched, and what it answered this time.
type bot struct {
	Container string          `json:"container"`
	State     string          `json:"state"` // Docker's: running, exited, restarting...
	Error     string          `json:"error,omitempty"`
	Stats     *botstats.Stats `json:"stats,omitempty"`
}

type watcher struct {
	docker string // the Docker API, read-only through docker-socket-proxy
	client *http.Client
}

type container struct {
	Names  []string          `json:"Names"`
	State  string            `json:"State"`
	Labels map[string]string `json:"Labels"`
}

func (c container) name() string {
	if len(c.Names) == 0 {
		return ""
	}
	return strings.TrimPrefix(c.Names[0], "/")
}

// bots lists every container carrying the label, stopped ones too so a dead bot still shows, and
// asks the running ones for their document, all at once.
func (w *watcher) bots() ([]bot, error) {
	cs, err := w.containers()
	if err != nil {
		return nil, err
	}
	out := make([]bot, len(cs))
	var wg sync.WaitGroup
	for i, c := range cs {
		out[i] = bot{Container: c.name(), State: c.State}
		if c.State != "running" {
			out[i].Error = "the container is " + c.State
			continue
		}
		wg.Add(1)
		go func(b *bot, u string) {
			defer wg.Done()
			s, err := w.fetch(u)
			if err != nil {
				b.Error = err.Error()
				return
			}
			b.Stats = s
		}(&out[i], c.Labels[statsLabel])
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Container < out[j].Container })
	return out, nil
}

func (w *watcher) containers() ([]container, error) {
	filters := url.QueryEscape(`{"label":["` + statsLabel + `"]}`)
	resp, err := w.client.Get(w.docker + "/containers/json?all=1&filters=" + filters)
	if err != nil {
		return nil, fmt.Errorf("docker does not answer")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker: HTTP %d", resp.StatusCode)
	}
	var out []container
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

func (w *watcher) fetch(u string) (*botstats.Stats, error) {
	if u == "" {
		return nil, fmt.Errorf("the %s label is empty", statsLabel)
	}
	resp, err := w.client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("its stats do not answer")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("its stats answer HTTP %d", resp.StatusCode)
	}
	var s botstats.Stats
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("its stats are not a /bot-stats document")
	}
	return &s, nil
}

func newWatcher(docker string) *watcher {
	return &watcher{docker: strings.TrimRight(docker, "/"), client: &http.Client{Timeout: 4 * time.Second}}
}
