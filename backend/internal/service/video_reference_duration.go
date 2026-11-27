package service

import (
	"encoding/base64"
	"encoding/binary"
	"math"
	"path"
	"strings"

	"github.com/tidwall/gjson"
)

// 参考视频计费：编辑/扩展的输入视频、OpenAI input_reference 视频等按时长每秒加收。
// 上传文件与 data: URL 在服务端读取 MP4/MOV（ISO BMFF）moov/mvhd 中的时长，
// 不信任客户端自报时长；远程 URL 不下载（避免 SSRF 与带宽放大），按单条上限计费。
const (
	// ReferenceVideoUnknownDurationSeconds 无法读取时长时单条参考视频的计费秒数。
	ReferenceVideoUnknownDurationSeconds = VideoBillingMaxDurationSeconds
	// referenceVideoMaxBillableSeconds 单条参考视频计费上限，防止异常元数据产生天价账单。
	referenceVideoMaxBillableSeconds = 3600
)

// ReferenceVideoInput 是一条参考视频；Data 非空表示上传内容，否则看 URL。
type ReferenceVideoInput struct {
	URL         string
	ContentType string
	FileName    string
	Data        []byte
}

var referenceVideoExtensions = map[string]struct{}{
	".mp4": {}, ".m4v": {}, ".mov": {}, ".webm": {}, ".mkv": {}, ".avi": {},
}

// ProbeMP4DurationSeconds 读取 MP4/MOV 中 moov/mvhd 记录的时长（秒）。
func ProbeMP4DurationSeconds(data []byte) (float64, bool) {
	moov, ok := findBMFFBox(data, "moov")
	if !ok {
		return 0, false
	}
	mvhd, ok := findBMFFBox(moov, "mvhd")
	if !ok || len(mvhd) < 4 {
		return 0, false
	}
	var timescale uint32
	var duration uint64
	switch mvhd[0] {
	case 0:
		if len(mvhd) < 20 {
			return 0, false
		}
		timescale = binary.BigEndian.Uint32(mvhd[12:16])
		raw := binary.BigEndian.Uint32(mvhd[16:20])
		if raw == math.MaxUint32 {
			return 0, false
		}
		duration = uint64(raw)
	case 1:
		if len(mvhd) < 32 {
			return 0, false
		}
		timescale = binary.BigEndian.Uint32(mvhd[20:24])
		duration = binary.BigEndian.Uint64(mvhd[24:32])
		if duration == math.MaxUint64 {
			return 0, false
		}
	default:
		return 0, false
	}
	if timescale == 0 || duration == 0 {
		return 0, false
	}
	seconds := float64(duration) / float64(timescale)
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return 0, false
	}
	return seconds, true
}

// findBMFFBox 在同一层级查找指定类型的 box，返回其 payload（不含 box 头）。
func findBMFFBox(data []byte, boxType string) ([]byte, bool) {
	for offset := 0; offset+8 <= len(data); {
		size := uint64(binary.BigEndian.Uint32(data[offset : offset+4]))
		typ := string(data[offset+4 : offset+8])
		header := uint64(8)
		switch size {
		case 0:
			size = uint64(len(data) - offset)
		case 1:
			if offset+16 > len(data) {
				return nil, false
			}
			size = binary.BigEndian.Uint64(data[offset+8 : offset+16])
			header = 16
		}
		if size < header || size > uint64(len(data)-offset) {
			return nil, false
		}
		if typ == boxType {
			return data[offset+int(header) : offset+int(size)], true
		}
		offset += int(size)
	}
	return nil, false
}

func decodeVideoDataURL(raw string) ([]byte, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return nil, false
	}
	comma := strings.IndexByte(raw, ',')
	if comma < 0 || !strings.HasSuffix(strings.ToLower(raw[5:comma]), ";base64") {
		return nil, false
	}
	payload := strings.TrimSpace(raw[comma+1:])
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(payload, "="))
	}
	if err != nil || len(decoded) == 0 {
		return nil, false
	}
	return decoded, true
}

func stripURLQuery(raw string) string {
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		return raw[:i]
	}
	return raw
}

// looksLikeVideoReference 根据 Content-Type、data: 前缀或扩展名判断是否为视频。
func looksLikeVideoReference(rawURL, contentType, fileName string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "video/") {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(rawURL))
	if strings.HasPrefix(lower, "data:") {
		return strings.HasPrefix(lower, "data:video/")
	}
	for _, candidate := range []string{fileName, stripURLQuery(lower)} {
		if _, ok := referenceVideoExtensions[strings.ToLower(path.Ext(strings.TrimSpace(candidate)))]; ok {
			return true
		}
	}
	return false
}

// isReferenceVideoUpload 判断 multipart 上传是否为视频（声明类型/扩展名，或能读出 mvhd 时长）。
func isReferenceVideoUpload(upload OpenAIImagesUpload) bool {
	if looksLikeVideoReference("", upload.ContentType, upload.FileName) {
		return true
	}
	_, ok := ProbeMP4DurationSeconds(upload.Data)
	return ok
}

func referenceVideoFromUpload(upload OpenAIImagesUpload) ReferenceVideoInput {
	return ReferenceVideoInput{ContentType: upload.ContentType, FileName: upload.FileName, Data: upload.Data}
}

// referenceVideoBillableSeconds 返回单条参考视频的计费秒数（向上取整，至少 1 秒）。
func referenceVideoBillableSeconds(input ReferenceVideoInput) int {
	data := input.Data
	if len(data) == 0 {
		decoded, ok := decodeVideoDataURL(input.URL)
		if !ok {
			return ReferenceVideoUnknownDurationSeconds
		}
		data = decoded
	}
	seconds, ok := ProbeMP4DurationSeconds(data)
	if !ok {
		return ReferenceVideoUnknownDurationSeconds
	}
	if seconds >= referenceVideoMaxBillableSeconds {
		return referenceVideoMaxBillableSeconds
	}
	billable := int(math.Ceil(seconds))
	if billable < 1 {
		billable = 1
	}
	return billable
}

// ReferenceVideoSecondsTotal 汇总一次请求所有参考视频的计费秒数。
func ReferenceVideoSecondsTotal(inputs []ReferenceVideoInput) int {
	total := 0
	for _, input := range inputs {
		total += referenceVideoBillableSeconds(input)
	}
	return total
}

// referenceVideoStrictFields 一定承载视频的字段；input_reference 可能是图片，需按类型/扩展名判断。
var referenceVideoStrictFields = []string{"video", "video_url", "input_video", "reference_video", "reference_videos"}

func isReferenceVideoFieldName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if i := strings.IndexByte(name, '['); i >= 0 {
		name = name[:i]
	}
	for _, field := range referenceVideoStrictFields {
		if name == field {
			return true
		}
	}
	return false
}

func referenceVideoURLFromJSON(value gjson.Result) string {
	if value.Type == gjson.String {
		return strings.TrimSpace(value.String())
	}
	if !value.IsObject() {
		return ""
	}
	for _, p := range []string{"url", "video_url.url", "video_url", "video.url", "data"} {
		if r := value.Get(p); r.Type == gjson.String {
			if u := strings.TrimSpace(r.String()); u != "" {
				return u
			}
		}
	}
	return ""
}

// collectJSONReferenceVideos 收集 JSON 请求中的输入/参考视频（字符串或 {url} 对象，支持数组）。
func collectJSONReferenceVideos(body []byte) []ReferenceVideoInput {
	if !gjson.ValidBytes(body) {
		return nil
	}
	var out []ReferenceVideoInput
	collect := func(value gjson.Result, requireVideo bool) {
		items := []gjson.Result{value}
		if value.IsArray() {
			items = value.Array()
		}
		for _, item := range items {
			rawURL := referenceVideoURLFromJSON(item)
			if rawURL == "" {
				continue
			}
			if requireVideo {
				typ := strings.ToLower(item.Get("type").String())
				ct := firstNonEmptyTrimmed(item.Get("content_type").String(), item.Get("mime_type").String())
				if typ != "video" && typ != "video_url" && !item.Get("video_url").Exists() && !looksLikeVideoReference(rawURL, ct, "") {
					continue
				}
			}
			out = append(out, ReferenceVideoInput{URL: rawURL})
		}
	}
	for _, field := range referenceVideoStrictFields {
		if value := gjson.GetBytes(body, field); value.Exists() {
			collect(value, false)
		}
	}
	// input_reference / images（straitapi 等 JSON 用 images 传参考）可能是图片，只计视频条目。
	for _, field := range []string{"input_reference", "images"} {
		if value := gjson.GetBytes(body, field); value.Exists() {
			collect(value, true)
		}
	}
	return out
}
