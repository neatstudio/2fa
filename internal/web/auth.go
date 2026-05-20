package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

const sessionCookieName = "twofa_session"

func GenerateToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func authenticated(r *http.Request, token string) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && secureEqual(cookie.Value, token) {
		return true
	}
	provided := r.URL.Query().Get("token")
	return provided != "" && secureEqual(provided, token)
}

func secureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func cleanTokenURL(r *http.Request) string {
	q := r.URL.Query()
	q.Del("token")
	path := r.URL.Path
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	if strings.TrimSpace(path) == "" {
		return "/"
	}
	return path
}
