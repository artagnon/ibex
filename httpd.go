package main

import (
	"html/template"
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
)

type Page struct {
	Title string
	Body string
}

type Message struct {
	Subject string
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := mux.Vars(r)["title"]
	p, _ := loadPage(title)
	t, err := template.ParseFiles("app/view.html")
	if err != nil {
		panic("Cannot open view.html" + err.Error())
	}
	t.Execute(w, p)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "app/index.html")
}

func loadPage(title string) (*Page, error) {
	bytes, _ := json.Marshal(Message{"foom"})
	return &Page{title, string(bytes)}, nil
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/view/{title}", viewHandler)
	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
