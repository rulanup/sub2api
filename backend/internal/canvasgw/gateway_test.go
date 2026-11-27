package canvasgw

import (
	"net/http/httptest"
	"testing"
)

func TestPublicOrigin(t *testing.T) {
	g := &Gateway{addr: ":8091"}
	r := httptest.NewRequest("GET", "http://192.168.1.8:8090/api/v1/canvas/session", nil)
	if got := g.PublicOrigin(r); got != "http://192.168.1.8:8091" {
		t.Fatalf("ip: got %q", got)
	}
	r.Header.Set("X-Forwarded-Host", "api.example.com")
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := g.PublicOrigin(r); got != "https://api.example.com:8091" {
		t.Fatalf("forwarded: got %q", got)
	}
	g.publicURL, g.publicHost = "https://canvas.example.com", "canvas.example.com"
	if got := g.PublicOrigin(r); got != "https://canvas.example.com" {
		t.Fatalf("public url: got %q", got)
	}
}

func TestAllowedOrigin(t *testing.T) {
	g := &Gateway{addr: ":8091", publicHost: "canvas.example.com"}
	r := httptest.NewRequest("GET", "http://127.0.0.1:8091/", nil)
	cases := map[string]bool{"127.0.0.1:8091": true, "canvas.example.com": true, "CANVAS.example.com": true, "evil.example": false}
	for host, want := range cases {
		if got := g.allowedOrigin(host, r); got != want {
			t.Fatalf("%s: got %v want %v", host, got, want)
		}
	}
	r.Header.Set("X-Forwarded-Host", "canvas2.example.com")
	if !g.allowedOrigin("canvas2.example.com", r) {
		t.Fatal("forwarded host should be allowed")
	}
}
