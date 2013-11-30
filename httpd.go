package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"log"
	"fmt"
	"code.google.com/p/go-imap/go1/imap"
	"time"
)

var c *imap.Client

func requestLogger(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
	handler.ServeHTTP(w, r)
    })
}

func inboxHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(listRecent(c, 20)))
}

func main() {
	c = initClient()
	c.Select("INBOX", true)
	defer c.Logout(30 * time.Minute)

	r := mux.NewRouter()
	r.HandleFunc("/inbox", inboxHandler)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("www")))
	http.Handle("/", r)
	http.ListenAndServe(":8080", requestLogger(http.DefaultServeMux))
}
