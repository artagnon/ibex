# ibex

A full-fledged mail client for GMail users featuring conversations,
labels, and server-side search. The IMAP fetcher and HTTP server are
written in Go; the client application is written using Angular.js,
Underscore and Bootstrap. The Go server establishes an IMAP connection
and emits JSON data for the client to consume.

![Screenshot](http://i.imgur.com/dui01HI.png)

## Hacking

Make sure you have a working Go and
[bower](https://github.com/bower/bower) installation; then do:

```
$ go get github.com/artagnon/ibex
$ cd $GOPATH/src/github.com/artagnon/ibex
$ bower install
```

Create a `gmail.credentails` with your email and password separated by
a newline. Then uncomment these two lines in httpd.go:

```go
	// r.HandleFunc("/Inbox.json", inboxHandler)
	// r.HandleFunc("/AllMail.json", allMailHandler)
```

Now run `ibex`, navigate to `http://localhost:8080` and wait
for the messages to load.

When you get annoyed with the amount of time it takes to render,
uncomment the `main` function in imap.go temporarily, `go build
imap.go`, then dump the JSON into `www/Inbox.json` and
`www/AllMail.json`. Let the corresponding lines in httpd.go remain
commented out.
