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
	"strconv"
	"encoding/json"
)

type Message struct {
	Subject string
	Date time.Time
	From *mail.Address
	ToList []*mail.Address
	CcList []*mail.Address
	ThreadID uint64
}

func listMessages (c *imap.Client, cmd *imap.Command) []byte {
	var messageList []*Message

	for _, rsp := range cmd.Data {
		header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
		threadid, _ := strconv.ParseUint(rsp.MessageInfo().Attrs["X-GM-THRID"].(string), 10, 0)
		if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
			date, _ := msg.Header.Date()
			fromList, _ := msg.Header.AddressList("From")
			toList, _ := msg.Header.AddressList("To")
			ccList, _ := msg.Header.AddressList("Cc")
			messageStruct := Message{msg.Header.Get("Subject"), date,
				fromList[0], toList, ccList, threadid}
			messageList = append(messageList, &messageStruct)
		}
	}
	cmd.Data = nil

	bytestring, _ := json.Marshal(messageList)
	return bytestring
}

func initClient () *imap.Client {
	var config *tls.Config

	// Connect to the server
	c, err := imap.DialTLS("imap.gmail.com", config)
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}

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

	return c
}

func main () {
	c := initClient()

	// Remember to log out and close the connection when finished
	defer c.Logout(30 * time.Second)

	// List all top-level mailboxes, wait for the command to finish
	fmt.Println("\nTop-level mailboxes:")
	cmd, _ := imap.Wait(c.List("", "*"))
	for _, rsp := range cmd.Data {
		fmt.Println("|--", rsp.MailboxInfo())
	}

	// Check for new unilateral server data responses
	for _, rsp := range c.Data {
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
	cmd, _ = imap.Wait(c.Fetch(set, "RFC822.HEADER", "X-GM-THRID"))

	// Process responses while the command is running
	fmt.Println("\nMost recent messages:")
	bytestring := listMessages(c, cmd)
	fmt.Println(string(bytestring))

	c.Select("[Gmail]/All Mail", true)
	fmt.Println("\nMessages with attachments:")
	set, _ = imap.NewSeqSet("")

	cmd, _ = imap.Wait(c.Search("X-GM-RAW", c.Quote("has:attachment")))
	results := cmd.Data[0].SearchResults()
	if set.AddNum(results[len(results) - 10:]...); set.Empty() {
		fmt.Println("Error: No time to complete search")
		return
	}
	cmd.Data = nil

	cmd, _ = imap.Wait(c.Fetch(set, "RFC822.HEADER", "X-GM-THRID"))
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
