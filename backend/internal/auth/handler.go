package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/platform/httpx"
)

const (
	maxUsernameLen = 100
	maxPasswordLen = 200
)

// Authenticator is what the handlers need from the auth service.
type Authenticator interface {
	Login(ctx context.Context, username, password string) (LoginResult, error)
	Me(ctx context.Context, c Claims) (Me, error)
}

// Handler serves the auth endpoints.
type Handler struct{ svc Authenticator }

// NewHandler returns a Handler for svc.
func NewHandler(svc Authenticator) *Handler { return &Handler{svc: svc} }

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.ValidationError(w, "request body must be a JSON object with username and password", nil)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if fields := validateLogin(req); len(fields) > 0 {
		httpx.ValidationError(w, "invalid login request", fields)
		return
	}

	res, err := h.svc.Login(r.Context(), req.Username, req.Password)
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
	case err != nil:
		httpx.Internal(w, err)
	default:
		httpx.JSON(w, http.StatusOK, loginResponse{
			AccessToken: res.AccessToken, TokenType: "Bearer", ExpiresIn: res.ExpiresIn,
		})
	}
}

func validateLogin(req loginRequest) map[string]string {
	fields := map[string]string{}
	if req.Username == "" {
		fields["username"] = "is required"
	} else if utf8.RuneCountInString(req.Username) > maxUsernameLen {
		fields["username"] = "must be at most 100 characters"
	}
	if req.Password == "" {
		fields["password"] = "is required"
	} else if utf8.RuneCountInString(req.Password) > maxPasswordLen {
		fields["password"] = "must be at most 200 characters"
	}
	return fields
}

type meResponse struct {
	User       meUser       `json:"user"`
	Dealership meDealership `json:"dealership"`
}

type meUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type meDealership struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Me handles GET /me. Authenticate must have run first.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFrom(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	m, err := h.svc.Me(r.Context(), claims)
	switch {
	case errors.Is(err, ErrUserNotFound):
		unauthorized(w)
	case err != nil:
		httpx.Internal(w, err)
	default:
		httpx.JSON(w, http.StatusOK, meResponse{
			User:       meUser{ID: m.UserID, Username: m.Username},
			Dealership: meDealership{ID: m.DealershipID, Name: m.DealershipName},
		})
	}
}
