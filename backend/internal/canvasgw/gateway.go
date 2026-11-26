// Package canvasgw 是无限画布的多租户网关：独立端口整站反代到画布进程。
//
// 流程：sub2api 前端用 JWT 调 POST /api/v1/canvas/session 换一张一次性票据，
// iframe 打开 <网关>/__canvas_login?ticket=...，网关验票后种下 HttpOnly 会话 cookie 并跳到 /；
// 之后每个请求（含 WebSocket）都校验 cookie，删掉客户端自带的 X-Canvas-User 再注入真实用户 ID。
package canvasgw

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cookieName = "canvas_session"
	loginPath  = "/__canvas_login"
	ticketTTL  = 60 * time.Second
	userHeader = "X-Canvas-User"
	keyEnv     = "CANVAS_GATEWAY_SIGN_KEY"
)

// Gateway 签发/校验画布票据与会话，并把已认证请求反代到画布。
type Gateway struct {
	addr      string
	publicURL string
	upstream  *url.URL
	signKey   []byte
	ttl       time.Duration
	proxy     *httputil.ReverseProxy
	mu        sync.Mutex
	used      map[string]time.Time
}

var (
	once sync.Once
	def  *Gateway
)

// Default 返回按环境变量初始化的网关；未设置 CANVAS_GATEWAY_ADDR 或配置非法时返回 nil。
func Default() *Gateway {
	once.Do(func() {
		addr := strings.TrimSpace(os.Getenv("CANVAS_GATEWAY_ADDR"))
		if addr == "" {
			return
		}
		gw, err := newGateway(addr)
		if err != nil {
			log.Printf("canvas gateway disabled: %v", err)
			return
		}
		def = gw
	})
	return def
}

func loadSignKey() ([]byte, error) {
	k := []byte(os.Getenv(keyEnv))
	if len(k) == 0 {
		k = make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return nil, err
		}
		log.Printf("canvas gateway: 未设置 %s，使用进程内随机签名密钥（重启后画布会话失效）", keyEnv)
		return k, nil
	}
	if len(k) < 32 {
		return nil, errors.New("canvas gateway 签名密钥至少 32 字节")
	}
	return k, nil
}

func newGateway(addr string) (*Gateway, error) {
	raw := strings.TrimSpace(os.Getenv("CANVAS_UPSTREAM"))
	if raw == "" {
		raw = "http://127.0.0.1:3000"
	}
	upstream, err := url.Parse(raw)
	if err != nil || upstream.Host == "" {
		return nil, fmt.Errorf("invalid CANVAS_UPSTREAM %q", raw)
	}
	k, err := loadSignKey()
	if err != nil {
		return nil, err
	}
	ttl := 12 * time.Hour
	if v := strings.TrimSpace(os.Getenv("CANVAS_SESSION_TTL")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("invalid CANVAS_SESSION_TTL %q", v)
		}
		ttl = d
	}
	gw := &Gateway{addr: addr, upstream: upstream, signKey: k, ttl: ttl, used: map[string]time.Time{}}
	gw.publicURL = strings.TrimRight(strings.TrimSpace(os.Getenv("CANVAS_GATEWAY_PUBLIC_URL")), "/")
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	base := proxy.Director
	proxy.Director = func(r *http.Request) {
		base(r)
		// 画布校验 Host 白名单，统一改成上游地址。
		r.Host = upstream.Host
	}
	proxy.FlushInterval = -1
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("canvas gateway upstream error: %s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "canvas unavailable", http.StatusBadGateway)
	}
	gw.proxy = proxy
	return gw, nil
}

// Addr 是网关监听地址。
func (g *Gateway) Addr() string { return g.addr }

// Upstream 是画布进程地址。
func (g *Gateway) Upstream() string { return g.upstream.String() }

// IssueTicket 签发 60 秒内有效、只能使用一次的登录票据。
func (g *Gateway) IssueTicket(uid int64) string {
	return g.sign("t", uid, time.Now().Add(ticketTTL))
}

// PublicOrigin 是浏览器访问网关的源；未配置 CANVAS_GATEWAY_PUBLIC_URL 时按请求主机名 + 网关端口推导。
func (g *Gateway) PublicOrigin(r *http.Request) string {
	if g.publicURL != "" {
		return g.publicURL
	}
	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	_, port, err := net.SplitHostPort(g.addr)
	if err != nil || port == "" {
		return scheme + "://" + host
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}

// LoginURL 是 iframe 应打开的网关登录地址。
func (g *Gateway) LoginURL(r *http.Request, ticket string) string {
	return g.PublicOrigin(r) + loginPath + "?ticket=" + url.QueryEscape(ticket)
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == loginPath {
		g.login(w, r)
		return
	}
	c, err := r.Cookie(cookieName)
	if err != nil {
		http.Error(w, "canvas session required", http.StatusUnauthorized)
		return
	}
	uid, _, _, ok := g.verify("c", c.Value)
	if !ok {
		http.Error(w, "canvas session invalid or expired", http.StatusUnauthorized)
		return
	}
	// 只接受网关自身同源页面发起的请求，并把 Origin 改写成上游地址，画布的同源校验才能通过。
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		r.Header.Set("Origin", g.upstream.Scheme+"://"+g.upstream.Host)
	}
	// 覆盖而非透传：客户端自带的身份头一律丢弃。
	r.Header.Del(userHeader)
	r.Header.Set(userHeader, strconv.FormatInt(uid, 10))
	g.proxy.ServeHTTP(w, r)
}

func (g *Gateway) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	uid, exp, nonce, ok := g.verify("t", r.URL.Query().Get("ticket"))
	if !ok || !g.consume(nonce, exp) {
		http.Error(w, "canvas ticket invalid, expired or already used", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    g.sign("c", uid, time.Now().Add(g.ttl)),
		Path:     "/",
		MaxAge:   int(g.ttl.Seconds()),
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// consume 记录已用票据，防止重放；顺带清理过期记录。
func (g *Gateway) consume(nonce string, exp time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	for k, t := range g.used {
		if now.After(t) {
			delete(g.used, k)
		}
	}
	if _, seen := g.used[nonce]; seen {
		return false
	}
	g.used[nonce] = exp
	return true
}

// 令牌格式：base64url(kind|uid|exp|nonce) + "." + base64url(HMAC-SHA256)。
func (g *Gateway) sign(kind string, uid int64, exp time.Time) string {
	nonce := make([]byte, 12)
	_, _ = rand.Read(nonce)
	payload := fmt.Sprintf("%s|%d|%d|%s", kind, uid, exp.Unix(), hex.EncodeToString(nonce))
	mac := hmac.New(sha256.New, g.signKey)
	mac.Write([]byte(payload))
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(payload)) + "." + enc.EncodeToString(mac.Sum(nil))
}

func (g *Gateway) verify(kind, value string) (uid int64, exp time.Time, nonce string, ok bool) {
	enc := base64.RawURLEncoding
	p, s, found := strings.Cut(value, ".")
	if !found {
		return
	}
	payload, err := enc.DecodeString(p)
	if err != nil {
		return
	}
	sig, err := enc.DecodeString(s)
	if err != nil {
		return
	}
	mac := hmac.New(sha256.New, g.signKey)
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return
	}
	parts := strings.Split(string(payload), "|")
	if len(parts) != 4 || parts[0] != kind {
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		return
	}
	unix, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}
	expiry := time.Unix(unix, 0)
	if time.Now().After(expiry) {
		return
	}
	return id, expiry, parts[3], true
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
