package utils

import (
	"fmt"
	"os"
	"strconv"
)

// MustGetenv attempts to lookup and return the value associated with the specified environment variable identifier, panic'ing if no value is associated with that identifier
func MustGetenv(env string) string {
	value, ok := os.LookupEnv(env)
	if !ok {
		panic(fmt.Sprintf("Failed to find environment variable with identifier: %s\n", env))
	}
	return value
}

// MustGetenvInt attempts to lookup and return the value associated with the specified environment variable identifier and cast it to an int,
//panic'ing if no value is associated with that identifier or it cannot be cast
func MustGetenvInt(env string) int {
	value := MustGetenv(env)
	intVal, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("Provided Environment variable for: %v can not be cast to int: %s\n", env, value))
	}
	return intVal
}

// MustGetenvIntNonZero attempts to lookup and return the value associated with the specified environment variable identifier and cast it to an int,
//panic'ing if no value is associated with that identifier, if it cannot be cast, or if once cast equals zero
func MustGetenvIntNonZero(env string) int {
	value := MustGetenvInt(env)
	if value == 0 {
		panic(fmt.Sprintf("Provided Environment variable equals 0 when explicitly not allowed: %s\n", env))
	}
	return value
}

// MustGetenvFloat attempts to lookup and return the value associated with the specified environment variable identifier and cast it to a float,
//panic'ing if no value is associated with that identifier or it cannot be cast
func MustGetenvFloat(env string) float64 {
	value := MustGetenv(env)
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(fmt.Sprintf("Provided Environment variable for: %v can not be cast to float: %s\n", env, value))
	}
	return floatVal
}

// MustGetenvFloatNonZero attempts to lookup and return the value associated with the specified environment variable identifier and cast it to a float,
//panic'ing if no value is associated with that identifier, if it cannot be cast, or if once cast equals zero
func MustGetenvFloatNonZero(env string) float64 {
	value := MustGetenvFloat(env)
	if value == 0 {
		panic(fmt.Sprintf("Provided Environment variable equals 0 when explicitly not allowed: %s\n", env))
	}
	return value
}

// EnvOrDefault fetches an environment variable value, or if not set returns the fallback value
func EnvOrDefault(key string, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
