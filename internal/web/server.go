package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/gouki/tools/2fa/internal/store"
)

type Options struct {
	StorePath string
	Port      int
	Addr      string
	Allows    []string
	Group     string
	Token     string
}

func Handler(st *store.Store, token string, allowValues []string) (http.Handler, error) {
	return HandlerWithGroup(st, token, allowValues, "")
}

func HandlerWithGroup(st *store.Store, token string, allowValues []string, defaultGroup string) (http.Handler, error) {
	prefixes, err := ParseAllowPrefixes(allowValues)
	if err != nil {
		return nil, err
	}
	a := &application{service: NewService(st), token: token, allowPrefixes: prefixes, defaultGroup: defaultGroup}
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/favicon.ico", a.handleFavicon)
	mux.HandleFunc("/api/accounts", a.handleAccounts)
	mux.HandleFunc("/api/accounts/", a.handleAccount)
	mux.HandleFunc("/api/events", a.handleEvents)
	return a.security(a.access(a.auth(mux))), nil
}

func ListenAndServe(ctx context.Context, opts Options, out io.Writer) error {
	if opts.Port == 0 {
		opts.Port = DefaultPort
	}
	if opts.Token == "" {
		token, err := GenerateToken()
		if err != nil {
			return err
		}
		opts.Token = token
	}
	localAddrs, err := LocalInterfaceAddrs()
	if err != nil {
		return err
	}
	targets, err := ResolveBindTargets(opts.Addr, opts.Port, localAddrs, opts.Allows)
	if err != nil {
		return err
	}
	handler, err := HandlerWithGroup(store.New(opts.StorePath), opts.Token, opts.Allows, opts.Group)
	if err != nil {
		return err
	}

	servers := make([]*http.Server, 0, len(targets))
	errCh := make(chan error, len(targets))
	for _, target := range targets {
		listener, err := net.Listen("tcp", target.ListenAddress())
		if err != nil {
			return err
		}
		server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
		servers = append(servers, server)
		fmt.Fprintf(out, "2fa web UI: %s\n", target.URL(opts.Token))
		go func() {
			err := server.Serve(listener)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
				return
			}
			errCh <- nil
		}()
	}
	fmt.Fprintln(out, "Listening only on localhost and allowed 100.64.0.0/10 addresses by default.")

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, server := range servers {
			_ = server.Shutdown(shutdownCtx)
		}
		return nil
	case err := <-errCh:
		for _, server := range servers {
			_ = server.Close()
		}
		return err
	}
}

type application struct {
	service       *Service
	token         string
	allowPrefixes []netip.Prefix
	defaultGroup  string
}

func (a *application) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (a *application) access(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !RemoteAllowed(r.RemoteAddr, a.allowPrefixes) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *application) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != "" && authenticated(r, a.token) {
			setSessionCookie(w, a.token)
			http.Redirect(w, r, cleanTokenURL(r), http.StatusFound)
			return
		}
		if !authenticated(r, a.token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *application) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, pageHTML)
}

func (a *application) handleFavicon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *application) handleAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := a.service.Snapshot(a.requestGroup(r), time.Now())
		writeJSON(w, response, err)
	case http.MethodPost:
		if !validMutation(r) {
			http.Error(w, "invalid mutation request", http.StatusBadRequest)
			return
		}
		var input AccountInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := a.service.Add(input, time.Now())
		writeJSON(w, view, err)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *application) handleAccount(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if name == "" || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	var err error
	name, err = url.PathUnescape(name)
	if err != nil {
		http.Error(w, "invalid account name", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		if !validMutation(r) {
			http.Error(w, "invalid mutation request", http.StatusBadRequest)
			return
		}
		var patch AccountPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		view, err := a.service.Update(name, patch, time.Now())
		writeJSON(w, view, err)
	case http.MethodDelete:
		if r.Header.Get("X-2FA-CSRF") != "1" {
			http.Error(w, "invalid mutation request", http.StatusBadRequest)
			return
		}
		if err := a.service.Delete(name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *application) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	group := a.requestGroup(r)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		response, err := a.service.Snapshot(group, time.Now())
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
		} else {
			data, _ := json.Marshal(response)
			fmt.Fprintf(w, "event: accounts\ndata: %s\n\n", data)
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *application) requestGroup(r *http.Request) string {
	if group := r.URL.Query().Get("group"); group != "" {
		return group
	}
	return a.defaultGroup
}

func writeJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func validMutation(r *http.Request) bool {
	return r.Header.Get("X-2FA-CSRF") == "1" && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
}
