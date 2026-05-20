package web

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gouki/tools/2fa/internal/store"
)

func TestHandlerAuthAndAccountFlow(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "accounts.json")
	handler, err := Handler(store.New(storePath), "token", nil)
	if err != nil {
		t.Fatal(err)
	}

	unauth := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	unauth.RemoteAddr = "127.0.0.1:1234"
	unauthResp := httptest.NewRecorder()
	handler.ServeHTTP(unauthResp, unauth)
	if unauthResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", unauthResp.Code)
	}

	login := httptest.NewRequest(http.MethodGet, "/?token=token", nil)
	login.RemoteAddr = "127.0.0.1:1234"
	loginResp := httptest.NewRecorder()
	handler.ServeHTTP(loginResp, login)
	if loginResp.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", loginResp.Code)
	}
	cookies := loginResp.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	post := authRequest(http.MethodPost, "/api/accounts", `{"name":"demo","secret":"JBSWY3DPEHPK3PXP","group":"work","note":"note"}`, cookies[0])
	post.Header.Set("Content-Type", "application/json")
	post.Header.Set("X-2FA-CSRF", "1")
	postResp := httptest.NewRecorder()
	handler.ServeHTTP(postResp, post)
	if postResp.Code != http.StatusOK {
		t.Fatalf("post failed: %d %s", postResp.Code, postResp.Body.String())
	}
	if strings.Contains(postResp.Body.String(), "JBSWY3DPEHPK3PXP") {
		t.Fatal("post response leaked secret")
	}

	list := authRequest(http.MethodGet, "/api/accounts?group=work", "", cookies[0])
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, list)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list failed: %d", listResp.Code)
	}
	if strings.Contains(listResp.Body.String(), "JBSWY3DPEHPK3PXP") {
		t.Fatal("list response leaked secret")
	}
	var listed AccountsResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Accounts) != 1 || listed.Accounts[0].Name != "demo" || listed.Accounts[0].Group != "work" {
		t.Fatalf("unexpected list response: %#v", listed)
	}

	patch := authRequest(http.MethodPatch, "/api/accounts/demo", `{"group":"game","note":"edited"}`, cookies[0])
	patch.Header.Set("Content-Type", "application/json")
	patch.Header.Set("X-2FA-CSRF", "1")
	patchResp := httptest.NewRecorder()
	handler.ServeHTTP(patchResp, patch)
	if patchResp.Code != http.StatusOK {
		t.Fatalf("patch failed: %d %s", patchResp.Code, patchResp.Body.String())
	}

	deleteReq := authRequest(http.MethodDelete, "/api/accounts/demo", "", cookies[0])
	deleteReq.Header.Set("X-2FA-CSRF", "1")
	deleteResp := httptest.NewRecorder()
	handler.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("delete failed: %d", deleteResp.Code)
	}
}

func TestHandlerRejectsRemoteAndMutationWithoutCSRF(t *testing.T) {
	handler, err := Handler(store.New(filepath.Join(t.TempDir(), "accounts.json")), "token", nil)
	if err != nil {
		t.Fatal(err)
	}
	blocked := httptest.NewRequest(http.MethodGet, "/?token=token", nil)
	blocked.RemoteAddr = "8.8.8.8:1234"
	blockedResp := httptest.NewRecorder()
	handler.ServeHTTP(blockedResp, blocked)
	if blockedResp.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", blockedResp.Code)
	}

	cookie := &http.Cookie{Name: sessionCookieName, Value: "token"}
	post := authRequest(http.MethodPost, "/api/accounts", `{"name":"demo","secret":"JBSWY3DPEHPK3PXP"}`, cookie)
	post.Header.Set("Content-Type", "application/json")
	postResp := httptest.NewRecorder()
	handler.ServeHTTP(postResp, post)
	if postResp.Code != http.StatusBadRequest {
		t.Fatalf("expected missing csrf to fail, got %d", postResp.Code)
	}
}

func TestEventsDoNotLeakSecrets(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "accounts.json")
	handler, err := Handler(store.New(storePath), "token", nil)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: sessionCookieName, Value: "token"}
	post := authRequest(http.MethodPost, "/api/accounts", `{"name":"demo","secret":"JBSWY3DPEHPK3PXP"}`, cookie)
	post.Header.Set("Content-Type", "application/json")
	post.Header.Set("X-2FA-CSRF", "1")
	handler.ServeHTTP(httptest.NewRecorder(), post)

	req := authRequest(http.MethodGet, "/api/events", "", cookie)
	ctx, cancel := contextWithTimeout(req, 150*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	line, err := bufio.NewReader(strings.NewReader(resp.Body.String())).ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "event: accounts") {
		t.Fatalf("expected accounts event, got %q err=%v", line, err)
	}
	if strings.Contains(resp.Body.String(), "JBSWY3DPEHPK3PXP") {
		t.Fatal("events leaked secret")
	}
}

func authRequest(method, target, body string, cookie *http.Cookie) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.AddCookie(cookie)
	return req
}

func contextWithTimeout(req *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(req.Context(), d)
}
