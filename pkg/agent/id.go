package agent

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func newHexID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// NewHexID is the exported 32-hex id used by HTTP handlers.
func NewHexID() string { return newHexID() }

func settingsID(userId int64) string {
	return fmt.Sprintf("agentset_%d", userId)
}
