package main

import (
	"database/sql"
	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
	"net/mail"
)

func insertThreadLabels(dbmap *gorp.DbMap, thread ThreadDb, labels []string) {
	var l LabelDb
	var m ThreadLabelMapper
	err := dbmap.SelectOne(&thread, "select * from thread where thread_id=?", thread.ThreadID)
	if err != nil {
		err = dbmap.Insert(&thread)
		checkErr(err, "Insert failed")
	}

	for _, label := range labels {
		err = dbmap.SelectOne(&l, "select * from label where label=?", label)
		if err != nil {
			l = newLabel(label)
			err = dbmap.Insert(&l)
			checkErr(err, "Insert failed")
		}

		err = dbmap.SelectOne(&m,
			"select * from thread_label_mapper where thread_id=? and label_id=?",
			thread.Id, l.Id)
		if err != nil {
			m = newThreadLabelMapper(thread.Id, l.Id)
			err = dbmap.Insert(&m)
			checkErr(err, "Insert failed")
		}
	}
}

func insertMessage(dbmap *gorp.DbMap, message MessageDb) {
	var m MessageDb
	err := dbmap.SelectOne(&m, "select * from message where message_id=?", message.MessageID)
	if err != nil {
		err = dbmap.Insert(&message)
		checkErr(err, "Insert failed")
	}
}

func retrieveThread(dbMap *gorp.DbMap, threadID string) (ThreadDb, error) {
	var thread ThreadDb
	err := dbmap.SelectOne(&thread, "select * from thread where thread_id=?", threadID)
	return thread, err
}

func retrieveMessages(dbMap *gorp.DbMap, thread ThreadDb) []*Message {
	var messagedbs []MessageDb
	var mappings []ThreadLabelMapper
	var label LabelDb
	var labels []string
	var messageList []*Message
	_, err := dbmap.Select(&messagedbs, "select * from message where thread_id=?", thread.Id)
	checkErr(err, "Failed to find message in thread")
	_, err = dbmap.Select(&mappings,
		"select * from thread_label_mapper where thread_id=?",
		thread.Id)
	/* err != nil indicates no labels for thread */
	for _, mapping := range mappings {
		err = dbmap.SelectOne(&label, "select * from label where id=?", mapping.LabelID)
		checkErr(err, "Inconsistency in mapper table")
		labels = append(labels, label.Label)
	}
	for _, messagedb := range messagedbs {
		addr, err := mail.ParseAddress(messagedb.From)
		checkErr(err, "Could not parse address")
		message := Message{
			Subject: thread.Subject,
			Date: messagedb.Date,
			From: addr,
			Labels: labels,
			ThreadID: thread.ThreadID,
			MessageID: messagedb.MessageID,
		}
		messageList = append(messageList, &message)
	}
	return messageList
}

type ThreadDb struct {
	Id       int64
	ThreadID string `db:"thread_id"`
	Subject  string
}

type LabelDb struct {
	Id      int64
	Label   string
}

type MessageDb struct {
	Id        int64
	ThreadID  int64  `db:"thread_id"`
	MessageID string `db:"message_id"`
	Date      time.Time
	From      string
}

type ThreadLabelMapper struct {
	ThreadID int64 `db:"thread_id"`
	LabelID  int64 `db:"label_id"`
}

func newThread(threadId string, subject string) ThreadDb {
	return ThreadDb{
		ThreadID: threadId,
		Subject: subject,
	}
}

func newLabel(label string) LabelDb {
	return LabelDb{
		Label: label,
	}
}

func newMessage(threadID string, messageID string, date time.Time, from string) MessageDb {
	var t ThreadDb
	err := dbmap.SelectOne(&t, "select * from thread where thread_id=?", threadID)
	checkErr(err, "Can't find thread corresponding to message")
	return MessageDb{
		ThreadID: t.Id,
		MessageID: messageID,
		Date: date,
		From: from,
	}
}

func newThreadLabelMapper(threadID int64, labelID int64) ThreadLabelMapper {
	return ThreadLabelMapper{
		ThreadID: threadID,
		LabelID:  labelID,
	}
}

func initDb(testing bool) *gorp.DbMap {
	dbName := "testing.db"
	if !testing {
		dbName = "mail.db"
	}
	db, err := sql.Open("sqlite3", dbName)
	checkErr(err, "sql.Open failed")

	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	// Enable foreign key support
	_, err = dbmap.Exec("pragma foreign_keys = ON")
	checkErr(err, "Failed to enable foreign key support")

	// add table for thread
	dbmap.AddTableWithName(ThreadDb{}, "thread").SetKeys(true, "Id").
		ColMap("ThreadID").SetUnique(true)

	// add table for label
	dbmap.AddTableWithName(LabelDb{}, "label").SetKeys(true, "Id").
		ColMap("Label").SetUnique(true)

	// add table for label
	dbmap.AddTableWithName(MessageDb{}, "message").SetKeys(true, "Id").
		ColMap("MessageID").SetUnique(true)

	// add many-to-many relationship table
	sql := `create table if not exists thread_label_mapper (
	thread_id integer, label_id integer,
	foreign key(thread_id) references thread(id) on delete cascade,
	foreign key(label_id) references label(id) on delete cascade
	);
	create index thread_index on thread_label_mapper(thread_id);
	create index label_index on thread_label_mapper(label_id);
	`

	_, err = dbmap.Exec(sql)
	checkErr(err, "Unable to create thread_label_mapper table")
	dbmap.AddTableWithName(ThreadLabelMapper{}, "thread_label_mapper")

	// create the thread and label tables
	err = dbmap.CreateTablesIfNotExists()
	checkErr(err, "Create tables failed")

	return dbmap
}

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}
