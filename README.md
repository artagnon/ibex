# ibex

A full-fledged mail client for GMail users featuring conversations,
labels, and server-side search. The IMAP fetcher and HTTP server are
written in Go; the client application is written using Angular.js,
Underscore and Bootstrap. The Go server establishes an IMAP connection
and emits JSON data for the client to consume.

This is pre-alpha software, so expect lots of bugs.

![Screenshot](http://i.imgur.com/dui01HI.png)

## Why another email client?

Existing open source clients don't even have feature parity with the
Gmail web interface (threading and search). Moreover, they require
that all emails be downloaded in advance: this is a non-starter.
Specifically, Mailr requires all emails to be downloaded first, IMAP
support is an afterthought in Mailpile.

ibex implements [Gmail IMAP
extensions](https://developers.google.com/gmail/imap_extensions):

1. Using X-GM-MSGID and X-GM-THRID, it groups individual emails into
   conversations. This is a very expensive operation, and is the
   reason mailboxes take a long time to load.

2. Using X-GM-LABELS, it fetches labels for individual
   messages. Conversation labels are then derived from individual
   message labels.

3. Using X-GM-RAW, it provides a way to do server-side search using
   Gmail's syntax. This can be really slow because IMAP SEARCH does
   not provide a way to LIMIT results.

The plan is to build a storage backend so mails are retrieved and
stored as necessary.

Existing clients like [sup](http://supmua.org) work with maildir/mbox,
but those formats have no way to represent conversations and labels.

## Running

Make sure you have a working Go and
[bower](https://github.com/bower/bower) installation; then do:

```
$ go get github.com/artagnon/ibex
$ cd $GOPATH/src/github.com/artagnon/ibex
$ bower install
```

Create a `~/.ibex/credentails` with your email and password separated by
a newline. Then run `ibex`, navigate to `http://localhost:8080` and
*wait* for the messages to load. Don't try a second operation before
the original one completes.

When you get annoyed with the amount of time it takes to render,
uncomment the `main` function in imap.go temporarily, `go build
imap.go`, then dump the JSON into `www/Inbox.json` and
`www/AllMail.json`. Comment out the corresponding endpoints in
httpd.go.
