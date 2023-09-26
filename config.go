package main

import (
	"encoding/json"
	"log"
	"os"
)

type Configuration struct {
	DatabasePath          string `json:"database_path"`
	DatabaseEncryptionKey string `json:"database_encryption_key"`

	UseTLS            bool   `json:"use_tls"`
	TLSCacheDirectory string `json:"tls_cache_directory"`
	AdminKey          string `json:"admin_key"`
}

func LoadConfiguration(path string) (Configuration, error) {
	var config Configuration
	configFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
		return config, err
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	return config, err
}
