package model

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateComponentIdentity validates values without consulting an account database.
func ValidateComponentIdentity(user *string, groups []string) error {
	if user != nil && *user != "" {
		parts := strings.Split(*user, ":")
		if len(parts) > 2 {
			return fmt.Errorf("user must be a name or UID, optionally followed by a group or GID")
		}
		for _, part := range parts {
			if !validIdentityToken(part) {
				return fmt.Errorf("user contains an empty or invalid identity")
			}
		}
	}
	for _, group := range groups {
		if !validIdentityToken(group) {
			return fmt.Errorf("group_add must contain nonempty group names or GIDs without whitespace or colons")
		}
	}
	return nil
}

func validIdentityToken(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char == ':' || unicode.IsSpace(char) || unicode.IsControl(char) {
			return false
		}
	}
	return true
}

// EffectiveComponentUser collapses an explicit image-default override for rendering.
func EffectiveComponentUser(user *string) *string {
	if user == nil || *user == "" {
		return nil
	}
	value := *user
	return &value
}
