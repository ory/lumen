package handlers

import (
	"errors"
	"net/http"
	"time"

	"example.com/portal/auth"
)

// ListRefunds handles GET /api/refunds.
func ListRefunds(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, listRefunds(sess.UserID))
}

// GetRefund handles GET /api/refunds.
func GetRefund(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, getRefund(sess.UserID, r.PathValue("id")))
}

// DeleteRefund handles DELETE /api/refunds.
func DeleteRefund(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, deleteRefund(sess.UserID, r.PathValue("id")))
}
