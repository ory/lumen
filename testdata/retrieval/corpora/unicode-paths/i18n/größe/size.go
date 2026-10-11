package groesse

import "strings"

// NormalizeSizeLabel maps free-form clothing labels to canonical sizes.
func NormalizeSizeLabel(label string) string {
	return strings.ToUpper(strings.TrimSpace(label))
}
