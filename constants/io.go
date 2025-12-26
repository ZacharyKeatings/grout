package constants

import "time"

const (
	DefaultBufferSize = 128 * 1024
	SmallBufferSize   = 64 * 1024
)

const (
	DefaultHTTPTimeout = 10 * time.Second
	LoginTimeout       = 6 * time.Second // Timeout for login attempts
	ValidationTimeout  = 3 * time.Second // Fast timeout for pre-flight connection checks
)
