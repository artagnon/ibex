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
	"sort"
	"strconv"
)

type Message struct {
	Subject string
	Date time.Time
	From *mail.Address
}

type MessageArray []*Message
type Conversations map[string]MessageArray

func (s MessageArray) Len() int {
	return len(s)
}
func (s MessageArray) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s MessageArray) Less(i, j int) bool {
	return s[i].Date.Before(s[j].Date)
}

func listMessages (c *imap.Client, cmd *imap.Command) MessageArray {
	var list MessageArray

	for _, rsp := range cmd.Data {
		header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
		if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
			date, _ := msg.Header.Date()
			fromList, _ := msg.Header.AddressList("From")
			messageStruct := Message{msg.Header.Get("Subject"), date, fromList[0]}
			list = append(list, &messageStruct)
		}
	}
	cmd.Data = nil

	return list
}

func threadSearch (c *imap.Client, threadID string) []*Message {
	set, _ := imap.NewSeqSet("")
	cmd, _ := imap.Wait(c.Search("X-GM-THRID", c.Quote(threadID)))
	results := cmd.Data[0].SearchResults()
	if set.AddNum(results...); set.Empty() {
		fmt.Println("Error: No search results")
		return nil
	}
	cmd.Data = nil

	cmd, _ = imap.Wait(c.Fetch(set, "RFC822.HEADER", "X-GM-THRID"))
	list := listMessages(c, cmd)
	return list
}

func listConversations (c *imap.Client, cmd *imap.Command) []byte {
	threads := make(map[string]bool)
	conversations := make(Conversations)
	conversationsD := make(Conversations)

	for _, rsp := range cmd.Data {
		threadid := rsp.MessageInfo().Attrs["X-GM-THRID"].(string)
		threads[threadid] = true
	}
	cmd.Data = nil

	c.Select("[Gmail]/All Mail", true)
	for threadid, _ := range threads {
		conversations[threadid] = threadSearch(c, threadid)
	}

	for _, value := range conversations {
		sort.Sort(MessageArray(value))
		newKey := value[len(value) - 1].Date
		conversationsD[strconv.FormatInt(newKey.Unix(), 10)] = value
	}

	bytestring, _ := json.Marshal(conversationsD)
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

func gmailSearch (c *imap.Client, searchString string, limit int) []byte {
	set, _ := imap.NewSeqSet("")
	cmd, _ := imap.Wait(c.Search("X-GM-RAW", c.Quote(searchString)))
	results := cmd.Data[0].SearchResults()
	var cut int
	if (len(results) < limit) { cut = 0; } else { cut = len(results) - limit; }
	if set.AddNum(results[cut:]...); set.Empty() {
		fmt.Println("Error: No time to complete search")
		return nil
	}
	cmd.Data = nil

	cmd, _ = imap.Wait(c.Fetch(set, "RFC822.HEADER", "X-GM-THRID"))
	bytestring := listConversations(c, cmd)
	return bytestring
}

func listRecent (c *imap.Client, limit uint32) []byte {
	// Fetch the headers of the 10 most recent messages
	set, _ := imap.NewSeqSet("")
	if c.Mailbox.Messages > limit {
		set.AddRange(c.Mailbox.Messages - limit, c.Mailbox.Messages)
	} else {
		set.Add("1:*")
	}

	cmd, _ := imap.Wait(c.Fetch(set, "RFC822.HEADER", "X-GM-THRID"))
	bytestring := listConversations(c, cmd)

	return bytestring
}

/*
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
	fmt.Println("\nMailbox status:\n", c.Mailbox)
	fmt.Println(string(listRecent(c, 20)))

	c.Select("[Gmail]/All Mail", true)
	fmt.Println("\nMessages with attachments:")
	fmt.Println(string(gmailSearch(c, "has:attachment", 20)))

	// Check command completion status
	if rsp, err := cmd.Result(imap.OK); err != nil {
		if err == imap.ErrAborted {
			fmt.Println("Fetch command aborted")
		} else {
			fmt.Println("Fetch error:", rsp.Info)
		}
	}
}
*/
