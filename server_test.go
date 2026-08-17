package gomemcache

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSetHandler(t *testing.T) {
	cs := NewCacheServer(NewCache())

	body := strings.NewReader(`{"key":"foo","value":"bar"}`)
	req := httptest.NewRequest(http.MethodPost, "/set", body)
	w := httptest.NewRecorder()

	cs.SetHandler(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("SetHandler() status = %v, want %v", w.Code, http.StatusAccepted)
	}

	value, found := cs.cache.Get("foo")
	if !found || value != "bar" {
		t.Errorf("cache.Get(foo) = %v,%v want bar,true", value, found)
	}
}

func TestSetHandlerWithTTL(t *testing.T) {
	cs := NewCacheServer(NewCache())

	body := strings.NewReader(`{"key":"foo","value":"bar","TTLSeconds":60}`)
	req := httptest.NewRequest(http.MethodPost, "/set", body)
	w := httptest.NewRecorder()

	cs.SetHandler(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("SetHandler() status = %v, want %v", w.Code, http.StatusAccepted)
	}

	item, found := cs.cache.items["foo"]
	if !found {
		t.Fatalf("expected item to be cached")
	}

	wantMin := time.Now().Add(59 * time.Second)
	if item.ExpiryTime.Before(wantMin) {
		t.Errorf("ExpiryTime = %v, want at least %v", item.ExpiryTime, wantMin)
	}
}

func TestSetHandlerInvalidBody(t *testing.T) {
	cs := NewCacheServer(NewCache())

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/set", body)
	w := httptest.NewRecorder()

	cs.SetHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SetHandler() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	if _, found := cs.cache.Get(""); found {
		t.Errorf("expected no entry to be cached on invalid body")
	}
}

func TestGetHandler(t *testing.T) {
	cache := NewCache()
	cache.Set("foo", "bar", time.Minute)
	cs := NewCacheServer(cache)

	req := httptest.NewRequest(http.MethodGet, "/get?key=foo", nil)
	w := httptest.NewRecorder()

	cs.GetHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetHandler() status = %v, want %v", w.Code, http.StatusOK)
	}

	want := `{"value":"bar"}` + "\n"
	if got := w.Body.String(); got != want {
		t.Errorf("GetHandler() body = %q, want %q", got, want)
	}
}

func TestGetHandlerMissingKey(t *testing.T) {
	cs := NewCacheServer(NewCache())

	req := httptest.NewRequest(http.MethodGet, "/get?key=missing", nil)
	w := httptest.NewRecorder()

	cs.GetHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetHandler() status = %v, want %v", w.Code, http.StatusNotFound)
	}
}
