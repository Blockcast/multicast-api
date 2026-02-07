//go:build tinygo

package api

// RToken is a stub read-lock token for TinyGo builds.
// Standard Go builds use xsync.RToken instead.
type RToken struct{}

// RWLocker describes the locking behavior required by shared transport structures.
// TinyGo version uses a simplified interface without xsync dependency.
type RWLocker interface {
	Lock()
	Unlock()
	RLock() *RToken
	RUnlock(t *RToken)
	TryRLock() (bool, *RToken)
	TryLock() bool
}
