package main

import (
	"encoding/json"
	"log"
	"os"
)

type Configuration struct {
	BaseURL      string `json:"base_url"`
	AbuseURL     string `json:"abuse_url"`
	DatabasePath string `json:"database_path"`
	AdminToken   string `json:"admin_token"`
	ListenOn     string `json:"listen_on"`
}

func LoadConfiguration() (Configuration, error) {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ", os.Args[0], " <config file>")
	}
	configPath := os.Args[1]
	var config Configuration
	configFile, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err)
		return config, err
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		log.Fatal(err)
		return config, err
	}
	if config.BaseURL == "" || config.AbuseURL == "" || config.DatabasePath == "" {
		log.Fatal("base_url, abuse_url, and database_path must be set in config file")
		return config, err
	}
	if config.ListenOn == "" {
		log.Println("listen_on not set in config file, defaulting to ':8080'")
		config.ListenOn = ":8080"
	}
	if config.AdminToken == "" {
		log.Println("AdminToken not set in config file, a random one will be generated")
		config.AdminToken = GenerateRandomString(32)
		log.Println("Generated AdminToken: ", config.AdminToken)
		return config, err
	}
	return config, err
}
