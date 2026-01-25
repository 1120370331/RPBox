package cache

import "time"

// TTL defines default cache TTLs by category.
var TTL = map[string]time.Duration{
	"user":         10 * time.Minute,
	"post:detail":  5 * time.Minute,
	"post:list":    30 * time.Second,
	"count:unread": 1 * time.Minute,
}
