package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlatanosVerdes/gonana/botstats"
)

// fakeDocker answers /containers/json with the given containers, as docker-socket-proxy does.
func fakeDocker(t *testing.T, cs []container) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/containers/json" || !strings.Contains(r.URL.Query().Get("filters"), statsLabel) {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(cs)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fakeBot(t *testing.T, s botstats.Stats) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(s)
	}))
	t.Cleanup(srv.Close)
	return srv
}

var torre = botstats.Stats{
	Platform: "telegram", Bot: "TorreFlightsBot", Name: "Torre",
	Users:   botstats.Users{Total: 12, Active24h: 3, Active7d: 8, New24h: 1},
	People:  []botstats.Person{{ID: "1", Name: "Ana"}},
	Traffic: botstats.Traffic{PerMinute: append(make([]int, 58), 7, 2), Messages24h: 120, Rejected24h: 3},
	Extra:   []botstats.Extra{{Label: "Flights followed", Value: "12"}},
}

func TestBotsAreFoundByLabelAndAsked(t *testing.T) {
	stats := fakeBot(t, torre)
	docker := fakeDocker(t, []container{
		{Names: []string{"/torre"}, State: "running", Labels: map[string]string{statsLabel: stats.URL + "/bot-stats"}},
		{Names: []string{"/wallabot"}, State: "exited", Labels: map[string]string{statsLabel: "http://wallabot:9180/bot-stats"}},
		{Names: []string{"/pol"}, State: "running", Labels: map[string]string{statsLabel: "http://127.0.0.1:1/bot-stats"}},
	})
	bots, err := newWatcher(docker.URL).bots()
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 3 {
		t.Fatalf("got %d bots", len(bots))
	}
	byName := map[string]bot{}
	for _, b := range bots {
		byName[b.Container] = b
	}
	if s := byName["torre"].Stats; s == nil || s.Users.Total != 12 || s.Extra[0].Value != "12" {
		t.Errorf("torre: %+v", byName["torre"])
	}
	if b := byName["wallabot"]; b.Stats != nil || !strings.Contains(b.Error, "exited") {
		t.Errorf("a stopped bot: %+v", b)
	}
	if b := byName["pol"]; b.Stats != nil || b.Error == "" {
		t.Errorf("a bot whose stats do not answer: %+v", b)
	}
}

func TestMetricsCarryTheFiguresButNoNames(t *testing.T) {
	var out strings.Builder
	writeMetrics(&out, []bot{{Container: "torre", State: "running", Stats: &torre}, {Container: "dead", State: "exited"}})
	m := out.String()
	for _, want := range []string{
		`muninn_bot_up{container="torre",platform="telegram",bot="TorreFlightsBot"} 1`,
		`muninn_bot_up{container="dead"} 0`,
		`muninn_bot_users{container="torre",platform="telegram",bot="TorreFlightsBot"} 12`,
		`muninn_bot_messages_last_minute{container="torre",platform="telegram",bot="TorreFlightsBot"} 7`,
		`muninn_bot_rejected_24h{container="torre",platform="telegram",bot="TorreFlightsBot"} 3`,
	} {
		if !strings.Contains(m, want) {
			t.Errorf("metrics lack %s", want)
		}
	}
	if strings.Contains(m, "Ana") || strings.Contains(m, `users{container="dead"`) {
		t.Error("metrics hold a name, or figures for a bot that did not answer")
	}
}
