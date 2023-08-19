package main

import (
	"flag"
	"log"

	badger "github.com/dgraph-io/badger/v3"
)

func main() {
	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	shortlink := flag.String("s", "shortlink", "Shortlink to add")
	target := flag.String("t", "target", "Target URL")
	delete_password := flag.String("p", "password", "Password to delete the shortlink")
	get := flag.Bool("get", false, "Get the shortlink")
	flag.Parse()
	if *get {
		rec, err := GetRecord(db, *shortlink)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Printf("Shortlink: %s\nRedirect: %s\n", rec.Shortlink, rec.RedirectTo)
		return
	}

	if *shortlink == "" || *target == "" || *delete_password == "" {
		log.Fatal("Missing required arguments")
		return
	}
	AddRecord(db, *shortlink, *target, *delete_password)

}
