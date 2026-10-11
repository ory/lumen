package auth

import (
	"errors"
	"net/http"
	"time"
)

// CookieName is the name of the session cookie set at login.
const CookieName = "portal_session"

var (
	ErrNoSession      = errors.New("auth: no session cookie")
	ErrSessionExpired = errors.New("auth: session expired")
)

// Session is the server-side state bound to a session cookie.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// Store loads sessions by id.
type Store interface {
	Get(id string) (*Session, error)
}

// ValidateSession loads the session referenced by the request cookie and
// rejects it when either the cookie or the stored session has expired.
func ValidateSession(r *http.Request, store Store, now time.Time) (*Session, error) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return nil, ErrNoSession
	}
	if !c.Expires.IsZero() && now.After(c.Expires) {
		return nil, ErrSessionExpired
	}
	s, err := store.Get(c.Value)
	if err != nil {
		return nil, err
	}
	if now.After(s.ExpiresAt) {
		return nil, ErrSessionExpired
	}
	return s, nil
}
