package handlers

import (
	"errors"
	"net/http"
	"time"

	"example.com/portal/auth"
)

// ListCoupons handles GET /api/coupons.
func ListCoupons(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, listCoupons(sess.UserID))
}

// GetCoupon handles GET /api/coupons.
func GetCoupon(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, getCoupon(sess.UserID, r.PathValue("id")))
}

// DeleteCoupon handles DELETE /api/coupons.
func DeleteCoupon(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.ValidateSession(r, sessions, time.Now())
	if errors.Is(err, auth.ErrSessionExpired) {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, deleteCoupon(sess.UserID, r.PathValue("id")))
}
