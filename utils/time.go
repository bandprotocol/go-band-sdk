package utils

import "time"

func MustParseDuration(t string) time.Duration {
	d, err := time.ParseDuration(t)
	if err != nil {
		panic(err)
	}
	return d
}
