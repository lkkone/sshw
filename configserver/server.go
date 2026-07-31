package configserver

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yinheli/sshw"
)

//go:embed web/*
var webFiles embed.FS

type ServerConfig struct {
	Store          *Store
	AdminUsername  string
	AdminPassword  string
	SessionSecret  string
	DefaultProfile string
	Logger         *log.Logger
}

type Server struct {
	store          *Store
	adminUsername  string
	adminPassword  string
	sessionKey     []byte
	defaultProfile string
	logger         *log.Logger
	mux            *http.ServeMux
	loginMu        sync.Mutex
	loginAttempts  map[string]*loginAttempt
}

type loginAttempt struct {
	Count        int
	BlockedUntil time.Time
	LastAttempt  time.Time
}

func NewServer(config ServerConfig) (*Server, error) {
	if config.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if strings.TrimSpace(config.AdminUsername) == "" || config.AdminPassword == "" {
		return nil, fmt.Errorf("admin username and password are required")
	}
	if config.SessionSecret == "" {
		return nil, fmt.Errorf("session secret is required")
	}
	if config.DefaultProfile == "" {
		config.DefaultProfile = "default"
	}
	if config.Logger == nil {
		config.Logger = log.Default()
	}
	if err := config.Store.EnsureProfile(context.Background(), config.DefaultProfile); err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte(config.SessionSecret))
	server := &Server{
		store:          config.Store,
		adminUsername:  config.AdminUsername,
		adminPassword:  config.AdminPassword,
		sessionKey:     key[:],
		defaultProfile: config.DefaultProfile,
		logger:         config.Logger,
		mux:            http.NewServeMux(),
		loginAttempts:  make(map[string]*loginAttempt),
	}
	server.routes()
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.securityHeaders(s.mux)
}

func (s *Server) routes() {
	content, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(content))))
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/logout", s.handleLogout)
	s.mux.HandleFunc("/api/session", s.handleSession)
	s.mux.Handle("/api/admin/state", s.requireAdmin(http.HandlerFunc(s.handleAdminState)))
	s.mux.Handle("/api/admin/render", s.requireAdmin(http.HandlerFunc(s.handleAdminRender)))
	s.mux.Handle("/api/admin/import", s.requireAdmin(http.HandlerFunc(s.handleAdminImport)))
	s.mux.Handle("/api/admin/draft", s.requireAdmin(http.HandlerFunc(s.handleAdminDraft)))
	s.mux.Handle("/api/admin/publish", s.requireAdmin(http.HandlerFunc(s.handleAdminPublish)))
	s.mux.Handle("/api/admin/restore", s.requireAdmin(http.HandlerFunc(s.handleAdminRestore)))
	s.mux.Handle("/api/admin/tokens", s.requireAdmin(http.HandlerFunc(s.handleAdminTokens)))
	s.mux.Handle("/api/admin/tokens/", s.requireAdmin(http.HandlerFunc(s.handleAdminToken)))
	s.mux.HandleFunc("/api/v1/sync/", s.handleSync)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "web application unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid request origin")
		return
	}
	ip := clientIP(r)
	if retry := s.loginRetryAfter(ip); retry > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
		writeError(w, http.StatusTooManyRequests, "too many login attempts")
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !secureEqual(input.Username, s.adminUsername) || !secureEqual(input.Password, s.adminPassword) {
		s.recordLoginFailure(ip)
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	s.clearLoginFailures(ip)
	s.setSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"username": s.adminUsername})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "sshw_admin",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestIsSecure(r),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if !s.authenticated(r) {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": s.adminUsername})
}

func (s *Server) handleAdminState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	profile := queryProfile(r, s.defaultProfile)
	state, err := s.store.State(r.Context(), profile)
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleAdminRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var document Document
	if err := decodeJSON(r, &document); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	yamlData, stats, err := RenderDocument(document)
	response := map[string]interface{}{
		"yaml":  string(yamlData),
		"stats": stats,
		"valid": err == nil,
	}
	if err != nil {
		response["error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleAdminImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, sshw.MaxConfigSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read configuration")
		return
	}
	if len(data) > sshw.MaxConfigSize {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("configuration exceeds %d bytes", sshw.MaxConfigSize))
		return
	}
	document, err := DocumentFromYAML(data)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	yamlData, stats, err := RenderDocument(document)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"document": document,
		"yaml":     string(yamlData),
		"stats":    stats,
	})
}

func (s *Server) handleAdminDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		methodNotAllowed(w, http.MethodPut)
		return
	}
	var document Document
	if err := decodeJSON(r, &document); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	yamlData, stats, err := RenderDocument(document)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	profile := queryProfile(r, s.defaultProfile)
	if err := s.store.SaveDraft(r.Context(), profile, document, yamlData); err != nil {
		s.internalError(w, err)
		return
	}
	sum := sha256.Sum256(yamlData)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"yaml":   string(yamlData),
		"sha256": fmt.Sprintf("%x", sum[:]),
		"stats":  stats,
	})
}

func (s *Server) handleAdminPublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var input struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	profile := queryProfile(r, s.defaultProfile)
	state, err := s.store.State(r.Context(), profile)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if _, err := sshw.ValidateConfig([]byte(state.YAML)); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "draft is not publishable: "+err.Error())
		return
	}
	published, err := s.store.Publish(r.Context(), profile, strings.TrimSpace(input.Note))
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"version": published.Version,
		"sha256":  published.SHA256,
		"note":    published.Note,
	})
}

func (s *Server) handleAdminRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var input struct {
		Version int `json:"version"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.Version <= 0 {
		writeError(w, http.StatusBadRequest, "version must be greater than zero")
		return
	}
	profile := queryProfile(r, s.defaultProfile)
	if err := s.store.RestoreVersionToDraft(r.Context(), profile, input.Version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "version not found")
			return
		}
		s.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"restoredVersion": input.Version})
}

func (s *Server) handleAdminTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 80 {
		writeError(w, http.StatusBadRequest, "device name is required and must be at most 80 characters")
		return
	}
	profile := queryProfile(r, s.defaultProfile)
	entry, token, err := s.store.CreateToken(r.Context(), profile, input.Name)
	if err != nil {
		s.internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"device": entry, "token": token})
}

func (s *Server) handleAdminToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}
	raw := strings.TrimPrefix(r.URL.Path, "/api/admin/tokens/")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	profile := queryProfile(r, s.defaultProfile)
	if err := s.store.RevokeToken(r.Context(), profile, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	profile := strings.TrimPrefix(r.URL.Path, "/api/v1/sync/")
	if profile == "" || strings.Contains(profile, "/") {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" || token == r.Header.Get("Authorization") {
		writeError(w, http.StatusUnauthorized, "device token required")
		return
	}
	valid, err := s.store.ValidateToken(r.Context(), profile, token)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if !valid {
		writeError(w, http.StatusUnauthorized, "invalid or revoked device token")
		return
	}
	config, err := s.store.Published(r.Context(), profile)
	if errors.Is(err, sqlErrNoRows()) {
		writeError(w, http.StatusNotFound, "no configuration has been published")
		return
	}
	if err != nil {
		s.internalError(w, err)
		return
	}
	etag := fmt.Sprintf(`"%s-v%d-%s"`, profile, config.Version, config.SHA256[:12])
	w.Header().Set("ETag", etag)
	w.Header().Set("X-SSHW-Version", strconv.Itoa(config.Version))
	w.Header().Set("X-SSHW-SHA256", config.SHA256)
	w.Header().Set("Cache-Control", "private, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(config.YAML)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Write(config.YAML)
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r) {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid request origin")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request) {
	expiry := time.Now().Add(12 * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%d", s.adminUsername, expiry)
	mac := hmac.New(sha256.New, s.sessionKey)
	mac.Write([]byte(payload))
	value := base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))))
	http.SetCookie(w, &http.Cookie{
		Name:     "sshw_admin",
		Value:    value,
		Path:     "/",
		Expires:  time.Unix(expiry, 0),
		MaxAge:   int((12 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestIsSecure(r),
	})
}

func (s *Server) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie("sshw_admin")
	if err != nil {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return false
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 3 || parts[0] != s.adminUsername {
		return false
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	payload := parts[0] + "|" + parts[1]
	mac := hmac.New(sha256.New, s.sessionKey)
	mac.Write([]byte(payload))
	return hmac.Equal(signature, mac.Sum(nil))
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) internalError(w http.ResponseWriter, err error) {
	s.logger.Printf("config server error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func (s *Server) loginRetryAfter(ip string) time.Duration {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	attempt := s.loginAttempts[ip]
	if attempt == nil || time.Now().After(attempt.BlockedUntil) {
		return 0
	}
	return time.Until(attempt.BlockedUntil)
}

func (s *Server) recordLoginFailure(ip string) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	now := time.Now()
	attempt := s.loginAttempts[ip]
	if attempt == nil || now.Sub(attempt.LastAttempt) > 15*time.Minute {
		attempt = &loginAttempt{}
		s.loginAttempts[ip] = attempt
	}
	attempt.Count++
	attempt.LastAttempt = now
	if attempt.Count >= 5 {
		attempt.BlockedUntil = now.Add(5 * time.Minute)
	}
}

func (s *Server) clearLoginFailures(ip string) {
	s.loginMu.Lock()
	delete(s.loginAttempts, ip)
	s.loginMu.Unlock()
}

func decodeJSON(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, sshw.MaxConfigSize+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func secureEqual(left, right string) bool {
	leftHash := sha256.Sum256([]byte(left))
	rightHash := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	expectedScheme := "http"
	if requestIsSecure(r) {
		expectedScheme = "https"
	}
	return origin == expectedScheme+"://"+r.Host
}

func requestIsSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func queryProfile(r *http.Request, fallback string) string {
	if profile := strings.TrimSpace(r.URL.Query().Get("profile")); profile != "" {
		return profile
	}
	return fallback
}

func sqlErrNoRows() error {
	return sql.ErrNoRows
}
