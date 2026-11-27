package service

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"mime/multipart"
	"testing"
)

func testBMFFBox(boxType string, payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(out[0:4], uint32(len(out)))
	copy(out[4:8], boxType)
	copy(out[8:], payload)
	return out
}

func testMP4WithDuration(version byte, timescale uint32, duration uint64) []byte {
	var mvhd []byte
	if version == 1 {
		mvhd = make([]byte, 112)
		mvhd[0] = 1
		binary.BigEndian.PutUint32(mvhd[20:24], timescale)
		binary.BigEndian.PutUint64(mvhd[24:32], duration)
	} else {
		mvhd = make([]byte, 100)
		binary.BigEndian.PutUint32(mvhd[12:16], timescale)
		binary.BigEndian.PutUint32(mvhd[16:20], uint32(duration))
	}
	out := testBMFFBox("ftyp", []byte("isomiso2mp41"))
	out = append(out, testBMFFBox("mdat", make([]byte, 64))...)
	return append(out, testBMFFBox("moov", testBMFFBox("mvhd", mvhd))...)
}

func TestProbeMP4DurationSeconds(t *testing.T) {
	valid := testMP4WithDuration(0, 1000, 8500)
	cases := []struct {
		name string
		data []byte
		want float64
		ok   bool
	}{
		{"mvhd v0", valid, 8.5, true},
		{"mvhd v1", testMP4WithDuration(1, 600, 3000), 5, true},
		{"zero timescale", testMP4WithDuration(0, 0, 100), 0, false},
		{"zero duration", testMP4WithDuration(0, 1000, 0), 0, false},
		{"no moov", testBMFFBox("ftyp", []byte("isom")), 0, false},
		{"truncated", valid[:len(valid)-10], 0, false},
		{"garbage", []byte("not a video"), 0, false},
		{"empty", nil, 0, false},
	}
	for _, tc := range cases {
		got, ok := ProbeMP4DurationSeconds(tc.data)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("%s: got (%v,%v), want (%v,%v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestReferenceVideoBillableSeconds(t *testing.T) {
	mp4 := testMP4WithDuration(0, 1000, 8500)
	dataURL := "data:video/mp4;base64," + base64.StdEncoding.EncodeToString(mp4)
	unknown := ReferenceVideoUnknownDurationSeconds
	cases := []struct {
		name  string
		input ReferenceVideoInput
		want  int
	}{
		{"upload rounds up", ReferenceVideoInput{Data: mp4}, 9},
		{"data url", ReferenceVideoInput{URL: dataURL}, 9},
		{"remote url uses cap", ReferenceVideoInput{URL: "https://cdn.example.com/a.mp4"}, unknown},
		{"unparsable upload uses cap", ReferenceVideoInput{Data: []byte("webm")}, unknown},
		{"sub-second is one", ReferenceVideoInput{Data: testMP4WithDuration(0, 1000, 200)}, 1},
		{"very long is capped", ReferenceVideoInput{Data: testMP4WithDuration(0, 1000, 7200*1000)}, referenceVideoMaxBillableSeconds},
	}
	for _, tc := range cases {
		if got := referenceVideoBillableSeconds(tc.input); got != tc.want {
			t.Fatalf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
	total := ReferenceVideoSecondsTotal([]ReferenceVideoInput{{Data: mp4}, {URL: "https://x/b.mov"}})
	if total != 9+unknown {
		t.Fatalf("total: got %d", total)
	}
	if got := ReferenceVideoSecondsTotal(nil); got != 0 {
		t.Fatalf("empty total: got %d", got)
	}
}

func TestLooksLikeVideoReference(t *testing.T) {
	cases := []struct {
		url, contentType, fileName string
		want                       bool
	}{
		{"https://x/a.mp4?sig=1", "", "", true},
		{"https://x/a.png", "", "", false},
		{"", "video/quicktime", "", true},
		{"data:video/mp4;base64,AA==", "", "", true},
		{"data:image/png;base64,AA==", "", "", false},
		{"", "", "clip.MOV", true},
		{"", "image/jpeg", "photo.jpg", false},
	}
	for _, tc := range cases {
		if got := looksLikeVideoReference(tc.url, tc.contentType, tc.fileName); got != tc.want {
			t.Fatalf("looksLikeVideoReference(%q,%q,%q)=%v", tc.url, tc.contentType, tc.fileName, got)
		}
	}
	if !isReferenceVideoUpload(OpenAIImagesUpload{FileName: "blob", Data: testMP4WithDuration(0, 1000, 3000)}) {
		t.Fatal("sniffed mp4 upload should count as video")
	}
	if isReferenceVideoUpload(OpenAIImagesUpload{FileName: "a.png", ContentType: "image/png", Data: []byte("png")}) {
		t.Fatal("png upload must not count as video")
	}
}

func testVideoForm(t *testing.T, field, name string, data []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", "m")
	_ = w.WriteField("prompt", "p")
	part, err := w.CreateFormFile(field, name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

func TestRequestParsersCountReferenceVideos(t *testing.T) {
	unknown := ReferenceVideoUnknownDurationSeconds
	clip := testMP4WithDuration(0, 1000, 4200)
	g := ParseGrokMediaRequest("application/json", []byte(`{"model":"grok-imagine-video","prompt":"p","video":{"url":"https://x/in.mp4"}}`))
	if g.ReferenceVideoSeconds != unknown {
		t.Fatalf("grok json video: %d", g.ReferenceVideoSeconds)
	}
	g = ParseGrokMediaRequest("application/json", []byte(`{"model":"grok-imagine-video","prompt":"p","image":{"url":"https://x/a.png"}}`))
	if g.ReferenceVideoSeconds != 0 {
		t.Fatalf("grok json image: %d", g.ReferenceVideoSeconds)
	}
	oa, err := ParseOpenAIVideoRequest("application/json", []byte(`{"model":"m","input_reference":"https://x/ref.mov"}`))
	if err != nil || oa.ReferenceVideoSeconds != unknown {
		t.Fatalf("openai json mov: %d %v", oa.ReferenceVideoSeconds, err)
	}
	oa, err = ParseOpenAIVideoRequest("application/json", []byte(`{"model":"m","input_reference":"https://x/ref.png"}`))
	if err != nil || oa.ReferenceVideoSeconds != 0 {
		t.Fatalf("openai json png: %d %v", oa.ReferenceVideoSeconds, err)
	}
	body, ct := testVideoForm(t, "input_reference", "clip.mp4", clip)
	oa, err = ParseOpenAIVideoRequest(ct, body)
	if err != nil || oa.ReferenceVideoSeconds != 5 || len(oa.Uploads) != 1 {
		t.Fatalf("openai multipart: %d uploads=%d %v", oa.ReferenceVideoSeconds, len(oa.Uploads), err)
	}
	body, ct = testVideoForm(t, "video", "clip.mp4", clip)
	if g = ParseGrokMediaRequest(ct, body); g.ReferenceVideoSeconds != 5 {
		t.Fatalf("grok multipart video: %d", g.ReferenceVideoSeconds)
	}
	body, ct = testVideoForm(t, "image", "clip.mp4", clip)
	if g = ParseGrokMediaRequest(ct, body); g.ReferenceVideoSeconds != 0 || len(g.Uploads) != 1 {
		t.Fatalf("grok multipart image: %d uploads=%d", g.ReferenceVideoSeconds, len(g.Uploads))
	}
}

func TestOpenAIVideoReferenceImagesAndRepeatedFields(t *testing.T) {
	unknown := ReferenceVideoUnknownDurationSeconds
	oa, err := ParseOpenAIVideoRequest("application/json", []byte(`{"model":"omni_flash-10s","prompt":"p","images":["https://x/a.jpg","https://x/b.mp4",{"url":"https://x/c","type":"video"}]}`))
	if err != nil || oa.ReferenceVideoSeconds != 2*unknown {
		t.Fatalf("json images: %d %v", oa.ReferenceVideoSeconds, err)
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", "m")
	_ = w.WriteField("input_reference", "https://x/a.png")
	_ = w.WriteField("input_reference", "https://x/b.mov")
	_ = w.WriteField("input_reference", "https://x/c.mp4")
	_ = w.Close()
	oa, err = ParseOpenAIVideoRequest(w.FormDataContentType(), buf.Bytes())
	if err != nil || oa.ReferenceVideoSeconds != 2*unknown {
		t.Fatalf("repeated input_reference: %d %v", oa.ReferenceVideoSeconds, err)
	}
}
