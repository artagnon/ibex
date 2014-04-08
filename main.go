package main

import "os"

func main () {
	if (len(os.Args) < 2) {
		httpMain()
	}
	switch os.Args[1] {
	case "http":
		httpMain()
	case "imap":
		imapMain()
	case "db":
		dbMain()
	}
}
