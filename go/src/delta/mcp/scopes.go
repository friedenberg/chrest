package mcp

import (
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

type AccessLevel string

const (
	AccessNone  AccessLevel = "none"
	AccessRead  AccessLevel = "read"
	AccessWrite AccessLevel = "write"
)

type Scope string

const (
	ScopeTabs       Scope = "tabs"
	ScopeWindows    Scope = "windows"
	ScopeBookmarks  Scope = "bookmarks"
	ScopeHistory    Scope = "history"
	ScopeManagement Scope = "management"
	ScopeState      Scope = "state"
)

type ScopeConfig map[Scope]AccessLevel

func DefaultScopes() ScopeConfig {
	return ScopeConfig{
		ScopeTabs:       AccessWrite,
		ScopeWindows:    AccessWrite,
		ScopeBookmarks:  AccessWrite,
		ScopeHistory:    AccessRead, // max is read
		ScopeManagement: AccessRead, // max is read
		ScopeState:      AccessWrite,
	}
}

func (sc ScopeConfig) Allows(scope Scope, required AccessLevel) bool {
	level, ok := sc[scope]
	if !ok {
		return false
	}

	return levelValue(level) >= levelValue(required)
}

func (sc ScopeConfig) AllowsAny(scopes []Scope, required AccessLevel) bool {
	for _, scope := range scopes {
		if sc.Allows(scope, required) {
			return true
		}
	}

	return false
}

func levelValue(level AccessLevel) int {
	switch level {
	case AccessNone:
		return 0
	case AccessRead:
		return 1
	case AccessWrite:
		return 2
	default:
		return 0
	}
}

func (sc ScopeConfig) MergeFrom(other map[string]string) {
	for k, v := range other {
		sc[Scope(k)] = AccessLevel(v)
	}
}

func (sc ScopeConfig) MergeFromFlags(flags []string) (err error) {
	for _, flag := range flags {
		parts := strings.SplitN(flag, ":", 2)
		if len(parts) != 2 {
			err = errors.Errorf("invalid scope format: %q (expected scope:level)", flag)
			return
		}

		scope := Scope(parts[0])
		level := AccessLevel(parts[1])

		if !isValidScope(scope) {
			err = errors.Errorf("unknown scope: %q", parts[0])
			return
		}

		if !isValidLevel(level) {
			err = errors.Errorf("unknown access level: %q (expected none, read, or write)", parts[1])
			return
		}

		sc[scope] = level
	}

	return
}

func isValidScope(s Scope) bool {
	switch s {
	case ScopeTabs, ScopeWindows, ScopeBookmarks, ScopeHistory, ScopeManagement, ScopeState:
		return true
	default:
		return false
	}
}

func isValidLevel(l AccessLevel) bool {
	switch l {
	case AccessNone, AccessRead, AccessWrite:
		return true
	default:
		return false
	}
}
