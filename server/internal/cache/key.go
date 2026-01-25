package cache

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// DefaultPrefix is the key namespace for RPBox cache entries.
const DefaultPrefix = "rpbox"

// Key builds a namespaced cache key.
func Key(parts ...string) string {
	return DefaultPrefix + ":" + strings.Join(parts, ":")
}

// VersionKey returns the key for a cache version counter.
func VersionKey(name string) string {
	return Key("cv", name)
}

// VersionedKey returns a versioned cache key.
func VersionedKey(name string, version int64, suffix string) string {
	if suffix == "" {
		return fmt.Sprintf("%s:%s:v%d", DefaultPrefix, name, version)
	}
	return fmt.Sprintf("%s:%s:v%d:%s", DefaultPrefix, name, version, suffix)
}

// HashKey generates a short hash for long filter strings.
func HashKey(value string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	return fmt.Sprintf("%x", h.Sum64())
}
