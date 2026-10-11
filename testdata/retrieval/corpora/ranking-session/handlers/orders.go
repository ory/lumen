package handlers

import (
	"errors"
	"net/http"
	"time"

	"example.com/portal/auth"
)

// ListOrders handles GET /api/orders.
func ListOrders(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, listOrders(sess.UserID))
}

// GetOrder handles GET /api/orders.
func GetOrder(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, getOrder(sess.UserID, r.PathValue("id")))
}

// DeleteOrder handles DELETE /api/orders.
func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, deleteOrder(sess.UserID, r.PathValue("id")))
}
