package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// avatars are the bots' profile photos, read from their public t.me page: the Bot API hands out a
// photo only behind a URL that carries the token, which must never reach a browser.
type avatars struct {
	base   string // https://t.me, replaced in tests
	client *http.Client

	mu    sync.Mutex
	cache map[string]avatar
}

type avatar struct {
	data    []byte
	kind    string
	fetched time.Time
}

const (
	avatarFresh  = 12 * time.Hour
	avatarRetry  = time.Hour // after a miss: a bot without a photo is not asked on every page load
	avatarMaxLen = 1 << 20
)

var (
	telegramName = regexp.MustCompile(`^[A-Za-z0-9_]{3,64}$`)
	ogImage      = regexp.MustCompile(`<meta property="og:image" content="([^"]+)"`)
)

func newAvatars() *avatars {
	return &avatars{base: "https://t.me", client: &http.Client{Timeout: 10 * time.Second}, cache: map[string]avatar{}}
}

// serve answers /api/avatar/telegram/<bot> with the photo, or 404 when there is none.
func (a *avatars) serve(w http.ResponseWriter, r *http.Request) {
	name, ok := strings.CutPrefix(r.URL.Path, "/api/avatar/telegram/")
	if !ok || !telegramName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	av := a.get(name)
	if len(av.data) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", av.kind)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(av.data)
}

func (a *avatars) get(name string) avatar {
	a.mu.Lock()
	av, ok := a.cache[name]
	a.mu.Unlock()
	if ok && (time.Since(av.fetched) < avatarFresh && len(av.data) > 0 || time.Since(av.fetched) < avatarRetry) {
		return av
	}
	fresh, err := a.fetch(name)
	if err != nil && len(av.data) > 0 {
		fresh = av // keep showing the last photo while t.me does not answer
	}
	fresh.fetched = time.Now()
	a.mu.Lock()
	a.cache[name] = fresh
	a.mu.Unlock()
	return fresh
}

func (a *avatars) fetch(name string) (avatar, error) {
	page, _, err := a.download(a.base + "/" + name)
	if err != nil {
		return avatar{}, err
	}
	m := ogImage.FindSubmatch(page)
	// Without a photo, the page offers Telegram's own logo.
	if m == nil || strings.Contains(string(m[1]), "telegram.org/img/") {
		return avatar{}, fmt.Errorf("no photo on the page")
	}
	img, kind, err := a.download(string(m[1]))
	if err != nil {
		return avatar{}, err
	}
	if !strings.HasPrefix(kind, "image/") {
		return avatar{}, fmt.Errorf("not an image: %s", kind)
	}
	return avatar{data: img, kind: kind}, nil
}

func (a *avatars) download(u string) ([]byte, string, error) {
	resp, err := a.client.Get(u)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, avatarMaxLen))
	return body, resp.Header.Get("Content-Type"), err
}
