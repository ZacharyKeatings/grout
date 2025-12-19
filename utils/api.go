package utils

import (
	"grout/romm"
	"time"
)

func GetRommClient(host romm.Host, timeout ...time.Duration) *romm.Client {
	opts := []romm.ClientOption{romm.WithBasicAuth(host.Username, host.Password)}
	if len(timeout) > 0 {
		opts = append(opts, romm.WithTimeout(timeout[0]))
	}
	return romm.NewClient(host.URL(), opts...)
}
