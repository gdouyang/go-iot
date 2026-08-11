package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-iot/pkg/api/web/session"
	"go-iot/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func setupWebMiniRedis(t *testing.T) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	restore := redis.SetClientForTest(client)
	t.Cleanup(func() {
		_ = client.Close()
		restore()
	})
}

func parseCookieHeader(setCookie string) (name, value string, maxAge string, attrs string) {
	// gsessionid=xxx; Path=/; Expires=...; Max-Age=3600; HttpOnly; SameSite=Lax
	parts := strings.Split(setCookie, ";")
	if len(parts) == 0 {
		return "", "", "", ""
	}
	nv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
	if len(nv) == 2 {
		name, value = nv[0], nv[1]
	}
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToLower(p), "max-age=") {
			maxAge = strings.TrimPrefix(p, "Max-Age=")
			maxAge = strings.TrimPrefix(maxAge, "max-age=")
			// case-insensitive trim
			if i := strings.Index(strings.ToLower(p), "max-age="); i >= 0 {
				maxAge = p[i+len("max-age="):]
			}
		}
	}
	attrs = setCookie
	return name, value, maxAge, attrs
}

func setCookieLines(w *httptest.ResponseRecorder) []string {
	// Go 1.x：同一 header 多值
	if vals := w.Result().Header.Values("Set-Cookie"); len(vals) > 0 {
		return vals
	}
	if sc := w.Header().Get("Set-Cookie"); sc != "" {
		return []string{sc}
	}
	return nil
}

func findSessionSetCookie(lines []string) string {
	for _, sc := range lines {
		if strings.HasPrefix(sc, SessionCookieName+"=") && !strings.Contains(sc, "Max-Age=0") {
			return sc
		}
	}
	return ""
}

func TestWriteSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctl := &RespController{}
	ctl.Init(w, r)

	s := &session.HttpSession{Sessionid: "abc123", ExpireSec: 1800}
	ctl.writeSessionCookie(s)

	lines := setCookieLines(w)
	require.NotEmpty(t, lines)
	// 应包含清理旧 Path + 写入 Path=/
	joined := strings.Join(lines, "\n")
	require.Contains(t, joined, "Path=/api")
	require.Contains(t, joined, "Max-Age=0")

	sc := findSessionSetCookie(lines)
	require.NotEmpty(t, sc)
	name, value, maxAge, full := parseCookieHeader(sc)
	require.Equal(t, "gsessionid", name)
	require.Equal(t, "abc123", value)
	require.Equal(t, "1800", maxAge)
	require.Contains(t, full, "HttpOnly")
	require.Contains(t, full, "SameSite=Lax")
	require.Contains(t, full, "Path=/")
}

func TestWriteSessionCookieNilSafe(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctl := &RespController{}
	ctl.Init(w, r)
	ctl.writeSessionCookie(nil)
	ctl.writeSessionCookie(&session.HttpSession{})
	require.Empty(t, setCookieLines(w))
}

func TestNewSessionWritesCookie(t *testing.T) {
	setupWebMiniRedis(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	ctl := NewController(w, r)

	s := ctl.NewSession(7200)
	require.NotEmpty(t, s.Sessionid)
	require.Equal(t, 7200, s.ExpireSec)

	sc := findSessionSetCookie(setCookieLines(w))
	require.Contains(t, sc, "gsessionid="+s.Sessionid)
	require.Contains(t, sc, "Max-Age=7200")
	require.Equal(t, s.Sessionid, r.Header.Get("x-access-token"))
}

func TestGetSessionRefreshesCookie(t *testing.T) {
	setupWebMiniRedis(t)

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/login", nil)
	ctl1 := NewController(w1, r1)
	s := ctl1.NewSession(3600)
	sid := s.Sessionid

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	r2.Header.Set("x-access-token", sid)
	ctl2 := NewController(w2, r2)

	got := ctl2.GetSession()
	require.NotNil(t, got)
	require.Equal(t, sid, got.Sessionid)

	sc := findSessionSetCookie(setCookieLines(w2))
	require.NotEmpty(t, sc, "GetSession should refresh cookie")
	require.Contains(t, sc, "gsessionid="+sid)
	require.Contains(t, sc, "Max-Age=3600")
	require.Contains(t, sc, "HttpOnly")
}

func TestGetSessionFromAuthorizationHeader(t *testing.T) {
	setupWebMiniRedis(t)
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/login", nil)
	s := NewController(w1, r1).NewSession(3600)

	// 前端默认：Authorization: <sessionId>，且可能仍带旧 cookie
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/user-info", nil)
	r2.Header.Set("Authorization", s.Sessionid)
	r2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "dead-old-session", Path: "/api"})
	ctl2 := NewController(w2, r2)

	got := ctl2.GetSession()
	require.NotNil(t, got)
	require.Equal(t, s.Sessionid, got.Sessionid)
}

func TestGetSessionFromCookieRefreshes(t *testing.T) {
	setupWebMiniRedis(t)

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/login", nil)
	ctl1 := NewController(w1, r1)
	s := ctl1.NewSession(120)
	sid := s.Sessionid

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	r2.AddCookie(&http.Cookie{Name: "gsessionid", Value: sid})
	ctl2 := NewController(w2, r2)

	got := ctl2.GetSession()
	require.NotNil(t, got)
	sc := findSessionSetCookie(setCookieLines(w2))
	require.Contains(t, sc, "gsessionid="+sid)
	require.Contains(t, sc, "Max-Age=120")
}

func TestGetSessionPrefersValidCookieWhenMultiple(t *testing.T) {
	setupWebMiniRedis(t)
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/login", nil)
	live := NewController(w1, r1).NewSession(120)

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	// 模拟两个同名 cookie：先旧后新（浏览器可能任意顺序）
	r2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "stale-id"})
	r2.AddCookie(&http.Cookie{Name: SessionCookieName, Value: live.Sessionid})
	ctl2 := NewController(w2, r2)

	got := ctl2.GetSession()
	require.NotNil(t, got)
	require.Equal(t, live.Sessionid, got.Sessionid)
}

func TestClearSessionCookies(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/logout", nil)
	ctl := &RespController{}
	ctl.Init(w, r)
	ctl.ClearSessionCookies()
	lines := setCookieLines(w)
	require.GreaterOrEqual(t, len(lines), 2)
	joined := strings.Join(lines, "\n")
	require.Contains(t, joined, "Max-Age=0")
	require.Contains(t, joined, "Path=/")
}

func TestGetSessionMissingNoCookie(t *testing.T) {
	setupWebMiniRedis(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	ctl := NewController(w, r)
	require.Nil(t, ctl.GetSession())
	require.Empty(t, setCookieLines(w))
}
