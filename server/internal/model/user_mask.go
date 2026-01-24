package model

import "strings"

// MaskedEmail returns a masked version of the user's email.
func (u *User) MaskedEmail() string {
	email := strings.TrimSpace(u.Email)
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}

	name := parts[0]
	domain := parts[1]
	if name == "" || domain == "" {
		return "***"
	}

	if len(name) <= 2 {
		return strings.Repeat("*", len(name)) + "@" + domain
	}

	return name[:2] + strings.Repeat("*", len(name)-2) + "@" + domain
}
