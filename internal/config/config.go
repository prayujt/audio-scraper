// Package config loads runtime configuration from the environment.
package config

import "github.com/caarlos0/env/v11"

// Config is the application configuration, populated from environment
// variables. It is parsed once at startup and passed through the composition
// root to the components that need it.
type Config struct {
	APIPort    string `env:"API_PORT" envDefault:"8080"`
	MusicHome  string `env:"MUSIC_HOME"`
	WorkerSize int    `env:"WORKER_SIZE" envDefault:"5"`

	// SubsonicURL is the base URL of a Subsonic-compatible server (Navidrome).
	// When empty, all Subsonic operations are skipped.
	SubsonicURL      string `env:"SUBSONIC_URL"`
	SubsonicUser     string `env:"SUBSONIC_USER"`
	SubsonicPassword string `env:"SUBSONIC_PASSWORD"`
}

// Load parses the environment into a Config.
func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
