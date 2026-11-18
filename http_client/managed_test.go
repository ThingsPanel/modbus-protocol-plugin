package httpclient

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestManagedPlatformAuthenticatesAndRejectsRedirect(t *testing.T) {
	var redirected atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer test-only" {
			t.Error("missing managed identity")
		}
		switch req.URL.Path {
		case "/device":
			_, _ = w.Write([]byte(`{"code":200}`))
		case "/redirect":
			http.Redirect(w, req, "/target", http.StatusTemporaryRedirect)
		default:
			redirected.Store(true)
		}
	}))
	defer server.Close()
	if _, err := newManagedPlatform(server.URL, ""); err == nil {
		t.Fatal("missing managed identity was accepted")
	}
	client, err := newManagedPlatform(server.URL, "test-only")
	if err != nil {
		t.Fatal(err)
	}
	var response struct{ Code int }
	if err := client.post("/device", nil, &response); err != nil || response.Code != 200 {
		t.Fatalf("managed response: %+v %v", response, err)
	}
	if err := client.post("/redirect", nil, &response); err == nil || redirected.Load() {
		t.Fatalf("managed request followed redirect: %t %v", redirected.Load(), err)
	}
}
