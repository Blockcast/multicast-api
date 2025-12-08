package api

import (
	"sync"

	"github.com/puzpuzpuz/xsync/v3"
)

// RWLocker describes the locking behavior required by shared transport structures.
type RWLocker interface {
	sync.Locker
	RLock() *xsync.RToken
	RUnlock(t *xsync.RToken)
	TryRLock() (bool, *xsync.RToken)
	TryLock() bool
}

