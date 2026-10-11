package handlers

import (
	"encoding/json"
	"net/http"

	"example.com/portal/auth"
)

var sessions auth.Store

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
