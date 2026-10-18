package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeTelegram(t *testing.T, pages map[string]string) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path == "/photo.jpg" {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte("jpeg bytes"))
			return
		}
		img, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if img == "photo" {
			img = srv.URL + "/photo.jpg"
		}
		w.Write([]byte(`<html><head><meta property="og:image" content="` + img + `"></head></html>`))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestAvatarComesFromThePublicPageAndIsCached(t *testing.T) {
	tg, hits := fakeTelegram(t, map[string]string{"/TorreFlightsBot": "photo", "/NoPhotoBot": "https://telegram.org/img/t_logo.png"})
	a := newAvatars()
	a.base = tg.URL

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		a.serve(rec, httptest.NewRequest("GET", "/api/avatar/telegram/TorreFlightsBot", nil))
		if rec.Code != 200 || rec.Body.String() != "jpeg bytes" || rec.Header().Get("Content-Type") != "image/jpeg" {
			t.Fatalf("got %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
		}
	}
	if *hits != 2 {
		t.Errorf("t.me asked %d times for two loads, want the page and the photo once", *hits)
	}

	rec := httptest.NewRecorder()
	a.serve(rec, httptest.NewRequest("GET", "/api/avatar/telegram/NoPhotoBot", nil))
	if rec.Code != 404 {
		t.Errorf("a bot whose page only has Telegram's logo answered %d", rec.Code)
	}
}

func TestAvatarOnlyAcceptsBotNames(t *testing.T) {
	a := newAvatars()
	a.base = "http://127.0.0.1:1"
	for _, p := range []string{"/api/avatar/telegram/..%2Fadmin", "/api/avatar/telegram/a/b", "/api/avatar/telegram/x"} {
		rec := httptest.NewRecorder()
		a.serve(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != 404 {
			t.Errorf("%s answered %d", p, rec.Code)
		}
	}
}
