package utils

import (
	"fmt"
	"os"
)

// MustGetenv attempts to lookup and return the value associated with the specified environment variable identifier, panic'ing if no value is associated with that identifier
func MustGetenv(env string) string {
	value, ok := os.LookupEnv(env)
	if !ok {
		panic(fmt.Sprintf("Failed to find environment variable with identifier: %s\n", env))
	}
	return value
}
