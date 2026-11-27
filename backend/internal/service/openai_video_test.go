package service

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

func buildVideoForm(t *testing.T, fields map[string]string, fileField string, file []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, k := range []string{"model", "prompt", "seconds", "size"} {
		if v, ok := fields[k]; ok {
			require.NoError(t, w.WriteField(k, v))
		}
	}
	if fileField != "" {
		fw, err := w.CreateFormFile(fileField, "ref.mp4")
		require.NoError(t, err)
		_, err = fw.Write(file)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes(), w.FormDataContentType()
}

func TestParseOpenAIVideoRequestMultipart(t *testing.T) {
	body, ct := buildVideoForm(t, map[string]string{"model": "zzdh-Minimax-h3-480p", "prompt": "a cat", "seconds": "6"}, "video[]", []byte{0, 1, 2, 3})
	info, err := ParseOpenAIVideoRequest(ct, body)
	require.NoError(t, err)
	require.Equal(t, "zzdh-Minimax-h3-480p", info.Model)
	require.Equal(t, "a cat", info.Prompt)
	require.Equal(t, 6, info.DurationSeconds)
	require.Len(t, info.Uploads, 1)
	require.Equal(t, "video[]", info.Uploads[0].FieldName)
}

func TestParseOpenAIVideoRequestJSON(t *testing.T) {
	info, err := ParseOpenAIVideoRequest("application/json", []byte(`{"model":"m","prompt":"p","seconds":5.2}`))
	require.NoError(t, err)
	require.Equal(t, 6, info.DurationSeconds, "小数秒向上取整")
	info, err = ParseOpenAIVideoRequest("application/json", []byte(`{"model":"m"}`))
	require.NoError(t, err)
	require.Equal(t, 0, info.DurationSeconds, "缺省时长留给计费处理")
	_, err = ParseOpenAIVideoRequest("application/json", []byte(`{"prompt":"p"}`))
	require.Error(t, err)
	_, err = ParseOpenAIVideoRequest("text/plain", []byte("hello"))
	require.Error(t, err)
}

func TestRewriteOpenAIVideoModelKeepsFiles(t *testing.T) {
	file := []byte("\x00\x01binary\r\n--not-a-boundary")
	body, ct := buildVideoForm(t, map[string]string{"model": "alias", "prompt": "p", "seconds": "6"}, "input_reference", file)
	out, err := rewriteOpenAIVideoModel(body, ct, "upstream-model")
	require.NoError(t, err)
	info, err := ParseOpenAIVideoRequest(ct, out)
	require.NoError(t, err)
	require.Equal(t, "upstream-model", info.Model)
	require.Equal(t, "p", info.Prompt)
	require.Len(t, info.Uploads, 1)
	require.Equal(t, file, info.Uploads[0].Data)
	jsonOut, err := rewriteOpenAIVideoModel([]byte(`{"model":"a","prompt":"p"}`), "application/json", "b")
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"b","prompt":"p"}`, string(jsonOut))
}

func TestBuildOpenAIVideoURL(t *testing.T) {
	u, err := buildOpenAIVideoURL("https://up.example.com/", OpenAIVideoEndpointCreate, "")
	require.NoError(t, err)
	require.Equal(t, "https://up.example.com/v1/videos", u)
	u, err = buildOpenAIVideoURL("https://up.example.com/v1", OpenAIVideoEndpointStatus, "task_1")
	require.NoError(t, err)
	require.Equal(t, "https://up.example.com/v1/videos/task_1", u)
	u, err = buildOpenAIVideoURL("https://up.example.com", OpenAIVideoEndpointContent, "task_1")
	require.NoError(t, err)
	require.Equal(t, "https://up.example.com/v1/videos/task_1/content", u)
	_, err = buildOpenAIVideoURL("https://up.example.com", OpenAIVideoEndpointStatus, "")
	require.Error(t, err)
	_, err = buildOpenAIVideoURL("https://up.example.com", OpenAIVideoEndpointStatus, "../x")
	require.Error(t, err)
}

func TestOpenAIVideoStatusAndID(t *testing.T) {
	require.True(t, openAIVideoStatusDone([]byte(`{"status":"completed"}`)))
	require.True(t, openAIVideoStatusDone([]byte(`{"code":0,"data":{"status":"SUCCESS"}}`)))
	require.False(t, openAIVideoStatusDone([]byte(`{"status":"in_progress"}`)))
	require.False(t, openAIVideoStatusDone([]byte(`{"status":"failed"}`)))
	require.Equal(t, "v1", openAIVideoResponseID([]byte(`{"id":"v1"}`)))
	require.Equal(t, "t2", openAIVideoResponseID([]byte(`{"data":{"task_id":"t2"}}`)))
	require.Equal(t, "", openAIVideoResponseID([]byte(`{"id":5}`)))
	require.Equal(t, "openai-video:abc", OpenAIVideoTaskKey(" abc "))
	require.Equal(t, "", OpenAIVideoTaskKey(""))
	require.Equal(t, "abc", openAIVideoUpstreamID("openai-video:abc"))
}

func TestOpenAIVideoEndpointClassification(t *testing.T) {
	require.True(t, OpenAIVideoEndpointCreate.IsGenerationRequest())
	require.True(t, OpenAIVideoEndpointCreate.RequiresRequestBody())
	require.True(t, OpenAIVideoEndpointStatus.IsVideoLookupRequest())
	require.True(t, OpenAIVideoEndpointContent.IsVideoLookupRequest())
	require.False(t, OpenAIVideoEndpointStatus.RequiresRequestBody())
	require.False(t, OpenAIVideoEndpointCreate.IsSeedance())
}
