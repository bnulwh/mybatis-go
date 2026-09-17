package orm

import "errors"

var (
	ErrInvalidDB        = errors.New("invalid db")
	ErrSafeUpdateBlocked = errors.New("safe-update blocked: UPDATE/DELETE without WHERE clause")
	ErrOptimisticLock   = errors.New("optimistic lock conflict: affected rows = 0")
)
