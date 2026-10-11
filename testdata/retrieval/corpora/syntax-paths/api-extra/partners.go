package apiextra

import "time"

// RotatePartnerKeys retires keys older than maxAge and issues replacements.
func RotatePartnerKeys(keys KeyStore, maxAge time.Duration) error {
	for _, k := range keys.Older(maxAge) {
		if err := keys.Retire(k); err != nil {
			return err
		}
	}
	return keys.Issue()
}
