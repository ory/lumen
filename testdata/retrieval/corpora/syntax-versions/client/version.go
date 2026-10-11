package client

import "fmt"

// minAgentVersion is the oldest monitoring agent release we accept.
const minAgentVersion = "3.2.1"

// checkAgentVersion rejects agents older than minAgentVersion.
func checkAgentVersion(v string) error {
	if v < minAgentVersion {
		return fmt.Errorf("agent %s is older than 3.2.1", v)
	}
	return nil
}
