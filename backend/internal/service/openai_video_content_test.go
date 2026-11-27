package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIVideoResultURL(t *testing.T) {
	cases := map[string]string{
		`{"video_url":"https://a/1.mp4","url":"https://a/2.mp4"}`: "https://a/1.mp4",
		`{"url":5,"content_url":"/v1/videos/t/content"}`:          "/v1/videos/t/content",
		`{"metadata":{"url":"https://a/3.mp4"}}`:                  "https://a/3.mp4",
		`{"data":[{"video_url":"https://a/4.mp4"}]}`:              "https://a/4.mp4",
		`{"videos":[{"url":"https://a/5.mp4"}]}`:                  "https://a/5.mp4",
		`{"data":{"url":"https://a/6.mp4"}}`:                      "https://a/6.mp4",
		`{"status":"completed"}`:                                  "",
	}
	for body, want := range cases {
		if got := openAIVideoResultURL([]byte(body)); got != want {
			t.Fatalf("%s: got %q want %q", body, got, want)
		}
	}
}

func TestOpenAIVideoDownloadClientBlocksPrivateHosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("video")) }))
	defer srv.Close()
	resp, err := newOpenAIVideoDownloadClient(nil).Get(srv.URL + "/a.mp4")
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("loopback video URL must be blocked")
	}
}
