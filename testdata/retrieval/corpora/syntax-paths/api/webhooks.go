package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SignWebhook computes the hex HMAC-SHA256 signature sent in X-Signature.
func SignWebhook(secret, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
