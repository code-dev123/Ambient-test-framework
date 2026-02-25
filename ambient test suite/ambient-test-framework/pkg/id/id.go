// pkg/id/id.go
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// Short returns a short random hex string (6 characters).
// Used as a unique suffix for test run resources (namespaces, labels, etc.).
func Short() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		panic("id.Short: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// Long returns a longer random hex string (16 characters).
func Long() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic("id.Long: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
