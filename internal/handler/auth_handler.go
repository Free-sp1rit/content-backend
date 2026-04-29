package handler

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type authService interface {
	Register(ctx context.Context, email, password string) (int64, error)
	Login(ctx context.Context, email, password, clientIP string) (string, error)
}

type AuthHandler struct {
	authService authService
}

func NewAuthHandler(authService authService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := h.authService.Register(r.Context(), req.Email, req.Password)
	if writeAuthServiceError(w, err) {
		return
	}

	res := RegisterResponse{ID: id}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password, clientIPFromRequest(r))
	if writeAuthServiceError(w, err) {
		return
	}

	res := LoginResponse{Token: token}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func clientIPFromRequest(r *http.Request) string {
	remoteIP := clientIPFromRemoteAddr(r.RemoteAddr)
	if trustedProxyIP(remoteIP) {
		if forwardedIP := firstForwardedIP(r.Header.Get("X-Forwarded-For")); forwardedIP != "" {
			return forwardedIP
		}
		if realIP := normalizedIP(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}

	return remoteIP
}

func clientIPFromRemoteAddr(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		if ip := normalizedIP(host); ip != "" {
			return ip
		}
		return host
	}

	if ip := normalizedIP(remoteAddr); ip != "" {
		return ip
	}

	return remoteAddr
}

func firstForwardedIP(header string) string {
	for _, part := range strings.Split(header, ",") {
		if ip := normalizedIP(part); ip != "" {
			return ip
		}
	}

	return ""
}

func trustedProxyIP(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	addr = addr.Unmap()
	return addr.IsLoopback() || addr.IsPrivate()
}

func normalizedIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	addr, err := netip.ParseAddr(value)
	if err != nil {
		return ""
	}

	return addr.Unmap().String()
}
