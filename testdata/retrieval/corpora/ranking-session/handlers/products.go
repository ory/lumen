package handlers

import (
	"errors"
	"net/http"
	"time"

	"example.com/portal/auth"
)

// ListProducts handles GET /api/products.
func ListProducts(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, listProducts(sess.UserID))
}

// GetProduct handles GET /api/products.
func GetProduct(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, getProduct(sess.UserID, r.PathValue("id")))
}

// DeleteProduct handles DELETE /api/products.
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, deleteProduct(sess.UserID, r.PathValue("id")))
}
