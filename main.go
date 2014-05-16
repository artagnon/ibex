package main

import "os"

func main () {
	if (len(os.Args) < 2) {
		httpMain(false)
	}
	switch os.Args[1] {
	case "http":
		httpMain(false)
	case "debug":
		httpMain(true)
	case "imap":
		imapMain()
	}
}
