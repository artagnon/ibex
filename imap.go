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
	"os/user"
	"github.com/coopernurse/gorp"
	"log"
)

var dbmap *gorp.DbMap
var useDbStore bool

type Message struct {
	Subject string
	Date time.Time
	From *mail.Address
	Flags []string
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

func extractQuotedList (listRaw []imap.Field) []string {
	var list []string
	for _, item := range listRaw {
		uqS, ok := imap.Unquote(item.(string))
		if ok == false { uqS = item.(string) }
		list = append(list, uqS)
	}
	return list
}

func listMessages (c *imap.Client, cmd *imap.Command) MessageArray {
	var list MessageArray

	for _, rsp := range cmd.Data {
		header := imap.AsBytes(rsp.MessageInfo().Attrs["BODY[HEADER]"])
		threadID, _ := rsp.MessageInfo().Attrs["X-GM-THRID"].(string)
		messageID, _ := rsp.MessageInfo().Attrs["X-GM-MSGID"].(string)
		flagsRaw := imap.AsList(rsp.MessageInfo().Attrs["FLAGS"])
		labelsRaw := imap.AsList(rsp.MessageInfo().Attrs["X-GM-LABELS"])
		flags := extractQuotedList(flagsRaw)
		labels := extractQuotedList(labelsRaw)
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
			fromList[0], flags, labels, threadID, messageID}

		// Insert into db
		thread := newThread(threadID, msg.Header.Get("Subject"))
		insertThread(dbmap, thread, labels, flags)
		message := newMessage(threadID, messageID, date, fromList[0].Name, fromList[0].Address)
		insertMessage(dbmap, message)

		list = append(list, &messageStruct)
	}
	cmd.Data = nil

	return list
}

func threadSearch (cmd *imap.Command) []*Message {
	set, _ := imap.NewSeqSet("")
	if _, err := cmd.Result(imap.OK); err != nil {
		fmt.Println(err.Error())
		return nil
	}
	results := cmd.Data[0].SearchResults()
	if set.AddNum(results...); set.Empty() {
		fmt.Println("Error: No search results")
		return nil
	}
	cmd.Data = nil

	cmd, err := imap.Wait(c.Fetch(set, "BODY.PEEK[HEADER]", "X-GM-THRID",
		"X-GM-MSGID", "FLAGS", "X-GM-LABELS"))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}

	return listMessages(c, cmd)
}

func listConversations (c *imap.Client, cmd *imap.Command) []byte {
	threadCmds := make(map[string]*imap.Command)
	conversations := make(Conversations)

	for _, rsp := range cmd.Data {
		threadid := rsp.MessageInfo().Attrs["X-GM-THRID"].(string)
		conversations[threadid] = nil
	}
	cmd.Data = nil

	/* Load threads from db, or retrieve over IMAP */
	for threadid, _ := range conversations {
		thread, err := retrieveThread(dbmap, threadid)
		if useDbStore && err == nil {
			conversations[threadid] = retrieveMessages(dbmap, thread)
		} else {
			selectMailbox(c, "[Gmail]/All Mail", true)
			fmt.Println("Fetching thread from IMAP")
			threadCmds[threadid], err = c.Search("X-GM-THRID", c.Quote(threadid))
			if err != nil {
				log.Fatalln("Search failed", err)
				delete(threadCmds, threadid)
			}
		}
	}
	for threadid, threadCmd := range threadCmds {
		conversations[threadid] = threadSearch(threadCmd)
	}

	/* Convert a hashtable keyed by threadID to a hashtable keyed
	/* by latest conversation date */
	var keys []string
	for key, _ := range conversations { keys = append(keys, key); }
	for _, key := range keys {
		value := conversations[key]
		if (value == nil) {
			fmt.Println("Error: conversation with key", key, "wasn't fetched")
			continue;
		}
		sort.Sort(MessageArray(value))
		newKey := value[len(value) - 1].Date
		delete(conversations, key)
		conversations[strconv.FormatInt(newKey.Unix(), 10)] = value
	}

	bytestring, _ := json.Marshal(conversations)
	return bytestring
}

func initClient (debug bool) *imap.Client {
	var config *tls.Config

	if (debug) {
		useDbStore = false
	} else {
		useDbStore = true
	}

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
	usr, _ := user.Current()
	credentialsFile := fmt.Sprintf("%s/%s", usr.HomeDir, ".ibex/credentials")
	if c.State() == imap.Login {
		b, err := ioutil.ReadFile(credentialsFile)
		if (err != nil) {
			fmt.Println(err.Error())
			return nil
		}
		userPass := strings.Split(string(b), "\n")
		c.Login(userPass[0], userPass[1])
	}

	// Initiate a db connection
	dbmap = initDb(false)

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

	cmd, err = imap.Wait(c.Fetch(set, "X-GM-THRID"))
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

	cmd, err := imap.Wait(c.Fetch(set, "X-GM-THRID"))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	bytestring := listConversations(c, cmd)

	return bytestring
}

func selectMailbox(c *imap.Client, name string, readonly bool) {
	if (c.Mailbox == nil || c.Mailbox.Name != name) {
		c.Select(name, readonly)
	}
}

func fetchMessage (c *imap.Client, messageID string) []byte {
	selectMailbox(c, "[Gmail]/All Mail", true)
	set, _ := imap.NewSeqSet("")
	qS := c.Quote(messageID)
	cmd, err := imap.Wait(c.UIDSearch("X-GM-MSGID", qS))
	if (err != nil) {
		fmt.Println(err.Error())
		return nil
	}
	results := cmd.Data[0].SearchResults()
	if len(results) != 1 {
		fmt.Println("Error: Could not get message for", messageID)
		return nil
	}
	set.AddNum(results[0])
	cmd.Data = nil

	var body []byte
	cmd, err = imap.Wait(c.UIDFetch(set, "BODY.PEEK[]"))
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

func imapMain () {
	c := initClient(false)
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
