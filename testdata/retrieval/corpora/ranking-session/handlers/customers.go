package handlers

import (
	"errors"
	"net/http"
	"time"

	"example.com/portal/auth"
)

// ListCustomers handles GET /api/customers.
func ListCustomers(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, listCustomers(sess.UserID))
}

// GetCustomer handles GET /api/customers.
func GetCustomer(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, getCustomer(sess.UserID, r.PathValue("id")))
}

// DeleteCustomer handles DELETE /api/customers.
func DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, deleteCustomer(sess.UserID, r.PathValue("id")))
}
