package config

import (
	"os"
	"sync"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type StoreConfig struct {
	SegmentsPerCPU         int `yaml:"segments_per_cpu"`
	CleanupIntervalSeconds int `yaml:"cleanup_interval_seconds"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type Config struct {
	Store  StoreConfig  `yaml:"store"`
	Server ServerConfig `yaml:"server"`
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig returns the singleton instance of Config
func GetConfig() *Config {

	//run once - singleton
	once.Do(func() {
		// Load .env file, ignore error if file doesn't exist
		godotenv.Load()

		// Get config path from environment
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml" // Default path
		}

		config, err := loadConfig(configPath)
		if err != nil {
			panic(err)
		}
		instance = config
	})
	return instance
}

// GetStoreConfig returns only the store configuration
func GetStoreConfig() *StoreConfig {
	return &GetConfig().Store
}

// GetServerConfig returns only the server configuration
func GetServerConfig() *ServerConfig {
	return &GetConfig().Server
}

func loadConfig(path string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// Set defaults if not specified
	if config.Store.SegmentsPerCPU == 0 {
		config.Store.SegmentsPerCPU = 4
	}
	if config.Store.CleanupIntervalSeconds == 0 {
		config.Store.CleanupIntervalSeconds = 1
	}
	if config.Server.Port == "" {
		config.Server.Port = "8090"
	}
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}

	return config, nil
}
