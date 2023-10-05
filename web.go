package main

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
)

type MainTemplateData struct {
	ErrorCreate      string
	ErrorDelete      string
	SuccessCreate    string
	SuccessDelete    string
	LinkToURL        string
	ShortlinkCreate  string
	ShortlinkDelete  string
	SuccessShortLink string
	AbuseUrl         string
}
type NotFoundData struct {
	ShortLink string
}

func SetupWebServer(db *badger.DB, config Configuration) {
	indexTemplate := template.Must(template.ParseFiles("index.html"))
	shortlinkNotFoundTemplate := template.Must(template.ParseFiles("shortlink404.html"))

	http.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, from Admin: %q", html.EscapeString(r.URL.Path))
	})
	http.HandleFunc("/s/", func(w http.ResponseWriter, r *http.Request) {
		shortlink := strings.SplitN(r.URL.Path, "/", 3)[2]
		if len(shortlink) == 0 {
			w.WriteHeader(404)
			shortlinkNotFoundTemplate.Execute(w, NotFoundData{ShortLink: shortlink})
			return
		}
		record, err := GetRecord(db, &shortlink)
		if err == ErrorShortlinkNotFound {
			w.WriteHeader(404)
			shortlinkNotFoundTemplate.Execute(w, NotFoundData{ShortLink: shortlink})
			return
		}
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
			return
		}
		http.Redirect(w, r, record.RedirectTo, http.StatusSeeOther)
	})
	http.HandleFunc("/create/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(400)
			return
		}
		linkToURL := r.Form.Get("url")
		shortlink := r.Form.Get("slug")
		password := r.Form.Get("password")
		passwordConfirm := r.Form.Get("password-confirm")
		_, err = url.ParseRequestURI(linkToURL)

		data := MainTemplateData{
			LinkToURL:       linkToURL,
			ShortlinkCreate: shortlink,
			AbuseUrl:        config.AbuseURL,
		}

		if err != nil {
			w.WriteHeader(400)
			data.ErrorCreate = "The URL is not valid."
			err = indexTemplate.Execute(w, data)
			if err != nil {
				w.WriteHeader(500)
				log.Println(err)
			}
			return
		}
		if password != passwordConfirm {
			w.WriteHeader(400)
			data.ErrorCreate = "Passwords do not match"
			err = indexTemplate.Execute(w, data)
			if err != nil {
				w.WriteHeader(500)
				log.Println(err)
			}
			return
		}
		if len(password) == 0 {
			w.WriteHeader(400)
			data.ErrorCreate = "Password cannot be empty"
			err = indexTemplate.Execute(w, data)
			if err != nil {
				w.WriteHeader(500)
				log.Println(err)
			}
			return
		}
		if len(shortlink) == 0 {
			shortlink = GenerateRandomShortlink(db)

		} else if !IsValidShortlink(&shortlink) {
			w.WriteHeader(400)
			data.ErrorCreate = "Shortlink is not valid. A shortlink may only contain the following characters: a-z, A-Z, 0-9, -, _ and be at least 3 characters long."
			err = indexTemplate.Execute(w, data)
			if err != nil {
				w.WriteHeader(500)
				log.Println(err)
			}
			return
		} else if DoesShortlinkExist(db, &shortlink) {
			w.WriteHeader(400)
			data.ErrorCreate = "Shortlink already exists, please choose another"
			err = indexTemplate.Execute(w, data)
			if err != nil {
				w.WriteHeader(500)
				log.Println(err)
			}
			return
		}
		if !AddRecord(db, &shortlink, &linkToURL, &password) {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(201)
		w.Header().Set("Location", "/")
		fullShortlink := config.BaseURL + "s/" + shortlink
		err = indexTemplate.Execute(w, MainTemplateData{SuccessCreate: "Shortlink created successfully: ", SuccessShortLink: fullShortlink})
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
		}
	})
	http.HandleFunc("/delete/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(400)
			return
		}
		shortlinkInput := strings.Split(r.Form.Get("slug"), "/")
		shortlink := shortlinkInput[len(shortlinkInput)-1]
		password := r.Form.Get("password")
		if password == config.AdminToken {
			err = DeleteRecordDisregardingPassword(db, &shortlink)
		} else {
			err = DeleteRecord(db, &shortlink, &password)
		}

		data := MainTemplateData{ShortlinkDelete: shortlink, AbuseUrl: config.AbuseURL}

		if err == ErrorShortlinkNotFound {
			w.WriteHeader(404)
			data.ErrorDelete = "The shortlink does not exist."
			err = indexTemplate.Execute(w, data)
			if err != nil {
				w.WriteHeader(500)
				log.Println(err)
			}
			return
		}
		if err == ErrorPasswordIncorrect {
			w.WriteHeader(401)
			data.ErrorDelete = "The deletion password is incorrect."
			err = indexTemplate.Execute(w, data)
			if err != nil {
				w.WriteHeader(500)
				log.Println(err)
			}
			return
		}
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
			return
		}
		data.SuccessDelete = "Shortlink deleted successfully: "
		err = indexTemplate.Execute(w, data)
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
		}
		w.Header().Set("Location", "/")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		err := indexTemplate.Execute(w, MainTemplateData{AbuseUrl: config.AbuseURL})
		if err != nil {
			w.WriteHeader(500)
			log.Println(err)
			return
		}
	})

}
