package main

import (
	"html/template"
	"io/ioutil"
	"net/http"
)

func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/view/"):]
	p, _ := loadPage(title)
	t, err := template.ParseFiles("/tmp/view.html")
	if err != nil {
		panic("Cannot open /tmp/view.html" + err.Error())
	}
	t.Execute(w, p)
}

type Page struct {
	Title string
	Body []byte
}

func (p *Page) save() error {
	filename := "/tmp/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := "/tmp/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func main() {
	http.HandleFunc("/view/", viewHandler)
	http.ListenAndServe(":8080", nil)
}
