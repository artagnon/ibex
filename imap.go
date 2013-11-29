package main

import (
	"code.google.com/p/go-imap/go1/imap"
	"fmt"
	"time"
	"bytes"
	"net/mail"
	"crypto/tls"
	"io/ioutil"
	"strings"
	"encoding/json"
)

type Message struct {
	Subject string
	Date time.Time
	From *mail.Address
	ToList []*mail.Address
	CcList []*mail.Address
}

func listMessages (c *imap.Client, cmd *imap.Command) []byte {
	var messageList []*Message

	for _, rsp := range cmd.Data {
		header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
		if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
			date, _ := msg.Header.Date()
			fromList, _ := msg.Header.AddressList("From")
			toList, _ := msg.Header.AddressList("To")
			ccList, _ := msg.Header.AddressList("Cc")
			messageStruct := Message{msg.Header.Get("Subject"), date,
				fromList[0], toList, ccList}
			messageList = append(messageList, &messageStruct)
		}
	}
	cmd.Data = nil

	for _, rsp := range c.Data {
		fmt.Println("Server data:", rsp)
	}
	c.Data = nil

	bytestring, _ := json.Marshal(messageList)
	return bytestring
}

func main () {
	var (
		cmd *imap.Command
		rsp *imap.Response
		config *tls.Config
	)

	// Connect to the server
	c, err := imap.DialTLS("imap.gmail.com", config)
	if (err != nil) {
		fmt.Println(err.Error())
		return
	}

	// Remember to log out and close the connection when finished
	defer c.Logout(30 * time.Second)

	// Print server greeting (first response in the unilateral server data queue)
	fmt.Println("Server says hello:", c.Data[0].Info)
	c.Data = nil

	// Enable encryption, if supported by the server
	if c.Caps["STARTTLS"] {
		c.StartTLS(nil)
	}

	// Authenticate
	if c.State() == imap.Login {
		b, err := ioutil.ReadFile("gmail-password.private")
		if (err != nil) { panic(err) }
		c.Login("artagnon@gmail.com", strings.TrimRight(string(b), " \r\n"))
	}

	// List all top-level mailboxes, wait for the command to finish
	cmd, _ = imap.Wait(c.List("", "%"))

	// Print mailbox information
	fmt.Println("\nTop-level mailboxes:")
	for _, rsp = range cmd.Data {
		fmt.Println("|--", rsp.MailboxInfo())
	}

	// Check for new unilateral server data responses
	for _, rsp = range c.Data {
		fmt.Println("Server data:", rsp)
	}
	c.Data = nil

	// Open a mailbox (synchronous command - no need for imap.Wait)
	c.Select("INBOX", true)
	fmt.Print("\nMailbox status:\n", c.Mailbox)

	// Fetch the headers of the 10 most recent messages
	set, _ := imap.NewSeqSet("")
	if c.Mailbox.Messages >= 10 {
		set.AddRange(c.Mailbox.Messages-9, c.Mailbox.Messages)
	} else {
		set.Add("1:*")
	}
	cmd, _ = imap.Wait(c.Fetch(set, "RFC822.HEADER"))

	// Process responses while the command is running
	fmt.Println("\nMost recent messages:")
	bytestring := listMessages(c, cmd)
	fmt.Println(string(bytestring))

	fmt.Println("\nMessages with attachments:")
	set, _ = imap.NewSeqSet("")

	cmd, _ = imap.Wait(c.Search("X-GM-RAW", c.Quote("has:attachment")))
	for _, rsp := range cmd.Data {
		set.AddNum(rsp.SearchResults()...)
	}
	cmd.Data = nil

	cmd, _ = imap.Wait(c.Fetch(set, "RFC822.HEADER"))
	bytestring = listMessages(c, cmd)
	fmt.Println(string(bytestring))

	// Check command completion status
	if rsp, err := cmd.Result(imap.OK); err != nil {
		if err == imap.ErrAborted {
			fmt.Println("Fetch command aborted")
		} else {
			fmt.Println("Fetch error:", rsp.Info)
		}
	}
}
