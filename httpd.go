package main

import (
	"html/template"
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
	"log"
)

type Page struct {
	Title string
	Body string
}

type Message struct {
	Subject string
}

func requestLogger(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
	handler.ServeHTTP(w, r)
    })
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := mux.Vars(r)["title"]
	p, _ := loadPage(title)
	t, err := template.ParseFiles("www/templates/view.html")
	if err != nil {
		panic("Cannot open view.html" + err.Error())
	}
	t.Execute(w, p)
}

func loadPage(title string) (*Page, error) {
	bytes, _ := json.Marshal(Message{"foom"})
	return &Page{title, string(bytes)}, nil
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/view/{title}", viewHandler)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("www")))
	http.Handle("/", r)
	http.ListenAndServe(":8080", requestLogger(http.DefaultServeMux))
}
