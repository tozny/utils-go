// Package cache contains concrete implementations of clients for interacting with cache datastores.
package cache

import (
	"crypto/tls"

	"github.com/redis/go-redis/v9"
)

// Config wraps configuration for a redis client.
type Config struct {
	Address            string
	Password           string
	ClusterModeEnabled bool
	TLSEnabled         bool
}

// NewClient returns a new redis client configured with the provided config.
func NewClient(config Config) redis.Cmdable {
	var client redis.Cmdable
	if config.ClusterModeEnabled {
		redisOpts := &redis.ClusterOptions{
			Addrs:    []string{config.Address},
			Password: config.Password,
		}
		if config.TLSEnabled {
			redisOpts.TLSConfig = &tls.Config{}
		}
		client = redis.NewClusterClient(redisOpts)
	} else {
		redisOpts := &redis.Options{
			Addr:     config.Address,
			Password: config.Password,
		}
		if config.TLSEnabled {
			redisOpts.TLSConfig = &tls.Config{}
		}
		client = redis.NewClient(redisOpts)
	}
	return client
}
