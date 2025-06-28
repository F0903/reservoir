package config

type Config struct {
	IgnoreNoCache bool
}

func Default() *Config {
	return &Config{
		IgnoreNoCache: true, // Since this is geared towards caching apt repositories, we aggressively cache responses.
	}
}
