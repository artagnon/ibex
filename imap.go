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
	Labels []string
	ThreadID string
	MessageID string
}

type MessageDetail struct {
	Body string
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
		header := imap.AsBytes(rsp.MessageInfo().Attrs["BODY[HEADER]"])
		threadID, _ := rsp.MessageInfo().Attrs["X-GM-THRID"].(string)
		messageID, _ := rsp.MessageInfo().Attrs["X-GM-MSGID"].(string)
		labelsRaw := imap.AsList(rsp.MessageInfo().Attrs["X-GM-LABELS"])
		var labels []string
		for _, label := range labelsRaw {
			uqS, ok := imap.Unquote(label.(string))
			if ok == false { uqS = label.(string) }
			labels = append(labels, uqS)
		}
		msg, err := mail.ReadMessage(bytes.NewReader(header))
		if (err != nil) {
			fmt.Println(err.Error())
			continue
		}
		date, err := msg.Header.Date()
		if (err != nil) {
			fmt.Println(err.Error())
			continue
		}
		fromList, err := msg.Header.AddressList("From")
		if (err != nil) {
			fmt.Println(err.Error())
			continue
		}
		messageStruct := Message{msg.Header.Get("Subject"), date,
			fromList[0], labels, threadID, messageID}
		list = append(list, &messageStruct)
	}
	cmd.Data = nil

	return list
}

func threadSearch (c *imap.Client, threadID string) []*Message {
	set, _ := imap.NewSeqSet("")
	cmd, err := imap.Wait(c.Search("X-GM-THRID", c.Quote(threadID)))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	results := cmd.Data[0].SearchResults()
	if set.AddNum(results...); set.Empty() {
		fmt.Println("Error: No search results")
		return nil
	}
	cmd.Data = nil

	cmd, err = imap.Wait(c.Fetch(set, "BODY[HEADER]", "X-GM-THRID",
		"X-GM-MSGID", "X-GM-LABELS"))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}

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

	for key, value := range conversations {
		if (value == nil) {
			fmt.Println("Error: conversation with key", key, "wasn't fetched")
			continue;
		}
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
		b, err := ioutil.ReadFile("gmail.credentials")
		if (err != nil) {
			fmt.Println(err.Error())
			return nil
		}
		userPass := strings.Split(string(b), "\n")
		c.Login(userPass[0], userPass[1])
	}

	return c
}

func gmailSearch (c *imap.Client, searchString string, limit int) []byte {
	set, _ := imap.NewSeqSet("")
	cmd, err := imap.Wait(c.Search("X-GM-RAW", c.Quote(searchString)))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	results := cmd.Data[0].SearchResults()
	var cut int
	if (len(results) < limit) { cut = 0; } else { cut = len(results) - limit; }
	if set.AddNum(results[cut:]...); set.Empty() {
		fmt.Println("Error: Empty search")
		return nil
	}
	cmd.Data = nil

	cmd, err = imap.Wait(c.Fetch(set, "BODY[HEADER]", "X-GM-THRID"))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	bytestring := listConversations(c, cmd)
	return bytestring
}

func listRecent (c *imap.Client, limit uint32) []byte {
	set, _ := imap.NewSeqSet("")
	if (c.Mailbox == nil) {
		fmt.Println("Error: No mailbox selected")
		return nil
	}
	if c.Mailbox.Messages > limit {
		set.AddRange(c.Mailbox.Messages - limit, c.Mailbox.Messages)
	} else {
		set.Add("1:*")
	}

	cmd, err := imap.Wait(c.Fetch(set, "BODY[HEADER]", "X-GM-THRID"))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	bytestring := listConversations(c, cmd)

	return bytestring
}

func fetchMessage (c *imap.Client, messageID string) []byte {
	c.Select("[Gmail]/All Mail", true)
	set, _ := imap.NewSeqSet("")
	qS := c.Quote(messageID)
	cmd, err := imap.Wait(c.UIDSearch("X-GM-MSGID", qS))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	result := cmd.Data[0].SearchResults()[0]
	set.AddNum(result)
	cmd.Data = nil

	var body []byte
	cmd, err = imap.Wait(c.UIDFetch(set, "BODY[]"))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	body = imap.AsBytes(cmd.Data[0].MessageInfo().Attrs["BODY[]"])
	cmd.Data = nil

	bytestring, err := json.Marshal(MessageDetail{string(body)})
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	return bytestring
}

/*
func main () {
	c := initClient()
	if (c == nil) { return }

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
	fmt.Println("\nMessages in All Mail:")
	fmt.Println(string(listRecent(c, 20)))
	// fmt.Println(string(gmailSearch(c, "has:attachment", 20)))

	fmt.Println(string(fetchMessage(c, "1452889506408166778")));

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
