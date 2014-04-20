# ibex

A full-fledged mail client for GMail users featuring conversations,
labels, and server-side search. Server-client architecture, where
multiple clients interact with a REST endpoint.

This is pre-alpha software, so expect lots of bugs.

![Screenshot](http://i.imgur.com/dui01HI.png)

## Why another email client?

1. Many don't even have feature parity with the Gmail web interface
   (conversations, labels, and search).

2. For threading and search functionality, other clients require all
   emails to be downloaded in advance.

3. Many clients don't feature sync-back: changes made on the client
   won't reflect in the Gmail web interface, making it impossible to
   use the web interface/ mobile client.

Gnus and mutt don't do conversations to begin with. Mailr, Mailpile,
[sup](http://supmua.org) require all emails to be downloaded in
advance, so they can be indexed.

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
*wait* for the messages to load.

## Clients

The debug client application is written using Angular.js, Underscore
and Bootstrap.

The Ncurses client, takin, is currently being written.

## Technical details

ibex implements [Gmail IMAP
extensions](https://developers.google.com/gmail/imap_extensions):

1. Using X-GM-MSGID and X-GM-THRID, it groups individual emails into
   conversations. This is a very expensive operation, and is the
   reason mailboxes take a long time to load.

2. Using X-GM-LABELS, it fetches labels for individual
   messages. Conversation labels are then derived from individual
   message labels.

3. Using X-GM-RAW, it provides a way to do server-side search using
   Gmail's syntax.

