package ports

import "time"

// Clock is injectable time for tests and future audit data.
type Clock interface {
	Now() time.Time
}
