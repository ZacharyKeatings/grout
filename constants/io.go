package constants

import "time"

const (
	DefaultBufferSize = 128 * 1024
	SmallBufferSize   = 64 * 1024
)

const (
	DefaultHTTPTimeout = 10 * time.Second
	LoginTimeout       = 15 * time.Second
)
