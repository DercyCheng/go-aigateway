package handlers

import (
	"time"
)

// generateID generates a unique ID based on timestamp
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + string(rune(time.Now().UnixNano()%1000))
}
