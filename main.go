package main

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
)

type MainTemplateData struct {
	ErrorCreate   string
	ErrorDelete   string
	SuccessCreate string
	SuccessDelete string
	AbuseUrl      string
}

func main() {
	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()
	http.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, from Admin: %q", html.EscapeString(r.URL.Path))
	})
	http.HandleFunc("/s/", func(w http.ResponseWriter, r *http.Request) {
		shortlink := strings.SplitN(r.URL.Path, "/", 3)[2]
		fmt.Fprintf(w, "Hello, redirecting: %s", html.EscapeString(shortlink))
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmplt, err := template.ParseFiles("index.html")
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
			return
		}

		data := MainTemplateData{
			ErrorCreate:   "The requested short link is already taken.",
			ErrorDelete:   "The deletion password is incorrect.",
			AbuseUrl:      "mailto:admin@example.com",
			SuccessCreate: "The short link was created successfully.",
			SuccessDelete: "The short link was deleted successfully.",
		}

		err = tmplt.Execute(w, data)
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
			return
		}
	})
	http.ListenAndServe(":8080", nil)

}
