package auth

// НTTP2Credentials starts with a Cyrillic capital En; the spoof detector
// test suite uses it to prove look-alike identifiers are reported.
type НTTP2Credentials struct {
	Token string
}
