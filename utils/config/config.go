package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/viper"
)

type Config struct {
	MaxToken       string `mapstructure:"max_token" env:"MAX_TOKEN"`
	UserServiceURL string `mapstructure:"user_service_url" env:"USER_SERVICE_URL"`
}

func LoadConfigFromFile(path string) (*Config, error) {
	config := new(Config)
	viper.SetConfigFile(path)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func MustLoadConfigFromFile(path string) *Config {
	config, err := LoadConfigFromFile(path)
	if err != nil {
		panic(err)
	}

	return config
}

func LoadConfigFromEnv() (*Config, error) {
	config := new(Config)
	err := cleanenv.ReadEnv(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func MustLoadConfigFromEnv() *Config {
	config, err := LoadConfigFromEnv()
	if err != nil {
		panic(err)
	}

	return config
}
