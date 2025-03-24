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

// TransitiveMustGetenv will (if a transitive dependnecy such as the value of another environment is set)
//
//	attempt to lookup the specified environment variable and return its string panicing if not provided
func TransitiveMustGetenv(env string, active bool) string {
	if !active {
		return ""
	}
	return MustGetenv(env)
}

// MustGetenvInt attempts to lookup and return the value associated with the specified environment variable identifier and cast it to an int,
// panic'ing if no value is associated with that identifier or it cannot be cast
func MustGetenvInt(env string) int {
	value := MustGetenv(env)
	intVal, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("Provided Environment variable for: %v can not be cast to int: %s\n", env, value))
	}
	return intVal
}

// TransitiveMustGetenvInt will (if a transitive dependnecy such as the value of another environment is set)
// attempt to lookup the specified environment variable parsed as an int and return its value panicing if not provided
func TransitiveMustGetenvInt(env string, active bool) int {
	if !active {
		return 0
	}
	return MustGetenvInt(env)
}

// MustGetenvIntNonZero attempts to lookup and return the value associated with the specified environment variable identifier and cast it to an int,
// panic'ing if no value is associated with that identifier, if it cannot be cast, or if once cast equals zero
func MustGetenvIntNonZero(env string) int {
	value := MustGetenvInt(env)
	if value == 0 {
		panic(fmt.Sprintf("Provided Environment variable equals 0 when explicitly not allowed: %s\n", env))
	}
	return value
}

// MustGetenvFloat attempts to lookup and return the value associated with the specified environment variable identifier and cast it to a float,
// panic'ing if no value is associated with that identifier or it cannot be cast
func MustGetenvFloat(env string) float64 {
	value := MustGetenv(env)
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(fmt.Sprintf("Provided Environment variable for: %v can not be cast to float: %s\n", env, value))
	}
	return floatVal
}

// TransitiveMustGetenvFloat will (if a transitive dependnecy such as the value of another environment is set)
// attempt to lookup the specified environment variable parsed as a float and return its value panicing if not provided
func TransitiveMustGetenvFloat(env string, active bool) float64 {
	if !active {
		return 0
	}
	return MustGetenvFloat(env)
}

// MustGetenvFloatNonZero attempts to lookup and return the value associated with the specified environment variable identifier and cast it to a float,
// panic'ing if no value is associated with that identifier, if it cannot be cast, or if once cast equals zero
func MustGetenvFloatNonZero(env string) float64 {
	value := MustGetenvFloat(env)
	if value == 0 {
		panic(fmt.Sprintf("Provided Environment variable equals 0 when explicitly not allowed: %s\n", env))
	}
	return value
}

// MustGetenvBool attempts to lookup and return the value associated with the specified environment variable identifier and cast it to a bool,
// panic'ing if no value is associated with that identifier, if it cannot be cast
func MustGetenvBool(env string) bool {
	value := MustGetenv(env)
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("Provided Environment variable for: %v could not be parsed to a bool: %s\n", env, value))
	}
	return boolVal
}

// TransitiveMustGetenvBool will (if a transitive dependnecy such as the value of another environment is set)
// attempt to lookup the specified environment variable parsed as a bool and return its value panicing if not provided
func TransitiveMustGetenvBool(env string, active bool) bool {
	if !active {
		return false
	}
	return MustGetenvBool(env)
}

// EnvOrDefault fetches an environment variable value, or if not set returns the fallback value
func EnvOrDefault(key string, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

// EnvOrDefaultBool fetches an environment variable value, or if not set returns the fallback boolean value.
func EnvOrDefaultBool(key string, fallback bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("Invalid boolean value for environment variable %q: %q. Remove env variable or set boolean value.", key, value))
	}
	return boolVal
}

// EnvOrDefaultInt fetches an environment variable as an integer.
// If the variable is not set or empty, it returns the fallback value.
func EnvOrDefaultInt(key string, fallback int) int {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return fallback
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("Invalid integer value for environment variable %q: %q. Remove env variable or set correct integer value.", key, value))
	}
	return intVal
}
