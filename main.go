package main

import (
	"log"
	"net/http"

	badger "github.com/dgraph-io/badger/v3"
)

func main() {
	config, err := LoadConfiguration()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
		return
	}

	db, err := badger.Open(badger.DefaultOptions(config.DatabasePath))
	if err != nil {
		log.Fatal("Failed to open db: ", err)
		return
	}
	defer db.Close()

	SetupWebServer(db, config)
	http.ListenAndServe(config.ListenOn, nil)
}
