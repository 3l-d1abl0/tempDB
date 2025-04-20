package config

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type StoreConfig struct {
	SegmentsPerCPU          int    `yaml:"segments_per_cpu"`
	CleanupIntervalSeconds  int    `yaml:"cleanup_interval_seconds"`
	WALFilePath             string `yaml:"wal_file_path"`
	SnapshotFilePath        string `yaml:"snapshot_file_path"`
	WALFlushIntervalSeconds int    `yaml:"wal_flush_interval_seconds"`
	SnapshotIntervalSeconds int    `yaml:"snapshot_interval_seconds"`
	WALMaxSizeBytes         int64  `yaml:"wal_max_size_bytes"`
	WALMaxFiles             int    `yaml:"wal_max_files"`
	WALDirectory            string `yaml:"wal_directory"`
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
	if config.Store.WALFilePath == "" {
		config.Store.WALFilePath = "wal.log"
	}
	if config.Store.SnapshotFilePath == "" {
		config.Store.SnapshotFilePath = "snapshot.db"
	}
	if config.Store.WALFlushIntervalSeconds == 0 {
		config.Store.WALFlushIntervalSeconds = 1
	}
	if config.Store.SnapshotIntervalSeconds == 0 {
		config.Store.SnapshotIntervalSeconds = 300 // 5 minutes
	}
	if config.Store.WALMaxSizeBytes == 0 {
		config.Store.WALMaxSizeBytes = 1024 * 1024 * 100 // 100MB default
	}
	if config.Store.WALMaxFiles == 0 {
		config.Store.WALMaxFiles = 5 // Keep 5 WAL files
	}
	if config.Store.WALDirectory == "" {
		config.Store.WALDirectory = filepath.Dir(config.Store.WALFilePath)
	}
	if config.Server.Port == "" {
		config.Server.Port = "8090"
	}
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}

	return config, nil
}
