package main

import (
	"database/sql"
	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
	"net/mail"
)

func insertThread(dbmap *gorp.DbMap, thread ThreadDb, labels []string, flags []string) {
	var l LabelDb
	var f FlagDb
	var tlm ThreadLabelMapper
	var tfm ThreadFlagMapper
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

		err = dbmap.SelectOne(&tlm,
			"select * from thread_label_mapper where thread_id=? and label_id=?",
			thread.Id, l.Id)
		if err != nil {
			tlm = newThreadLabelMapper(thread.Id, l.Id)
			err = dbmap.Insert(&tlm)
			checkErr(err, "Insert failed")
		}
	}

	for _, flag := range flags {
		err = dbmap.SelectOne(&f, "select * from flag where flag=?", flag)
		if err != nil {
			f = newFlag(flag)
			err = dbmap.Insert(&f)
			checkErr(err, "Insert failed")
		}

		err = dbmap.SelectOne(&tfm,
			"select * from thread_flag_mapper where thread_id=? and flag_id=?",
			thread.Id, f.Id)
		if err != nil {
			tfm = newThreadFlagMapper(thread.Id, f.Id)
			err = dbmap.Insert(&tfm)
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
	var tlms []ThreadLabelMapper
	var tfms []ThreadFlagMapper
	var label LabelDb
	var labels []string
	var flag FlagDb
	var flags []string
	var messageList []*Message
	_, err := dbmap.Select(&messagedbs, "select * from message where thread_id=?", thread.Id)
	checkErr(err, "Failed to find message in thread")

	/* Labels */
	_, err = dbmap.Select(&tlms,
		"select * from thread_label_mapper where thread_id=?",
		thread.Id)
	/* err != nil indicates no labels for thread */
	for _, mapping := range tlms {
		err = dbmap.SelectOne(&label, "select * from label where id=?", mapping.LabelID)
		checkErr(err, "Inconsistency in mapper table")
		labels = append(labels, label.Label)
	}

	/* Flags */
	_, err = dbmap.Select(&tfms,
		"select * from thread_flag_mapper where thread_id=?",
		thread.Id)
	/* err != nil indicates no flags for thread */
	for _, mapping := range tfms {
		err = dbmap.SelectOne(&flag, "select * from flag where id=?", mapping.FlagID)
		checkErr(err, "Inconsistency in mapper table")
		flags = append(flags, flag.Flag)
	}

	for _, messagedb := range messagedbs {
		addr := mail.Address{
			Name: messagedb.FromName,
			Address: messagedb.FromAddr,
		}
		checkErr(err, "Could not parse address")
		message := Message{
			Subject: thread.Subject,
			Date: messagedb.Date,
			From: &addr,
			Flags: flags,
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

type FlagDb struct {
	Id      int64
	Flag    string
}

type MessageDb struct {
	Id        int64
	ThreadID  int64  `db:"thread_id"`
	MessageID string `db:"message_id"`
	Date      time.Time
	FromName  string
	FromAddr  string
}

type ThreadLabelMapper struct {
	ThreadID int64 `db:"thread_id"`
	LabelID  int64 `db:"label_id"`
}

type ThreadFlagMapper struct {
	ThreadID int64 `db:"thread_id"`
	FlagID   int64 `db:"flag_id"`
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

func newFlag(flag string) FlagDb {
	return FlagDb{
		Flag: flag,
	}
}

func newMessage(threadID string, messageID string, date time.Time, fromName string, fromAddr string) MessageDb {
	var t ThreadDb
	err := dbmap.SelectOne(&t, "select * from thread where thread_id=?", threadID)
	checkErr(err, "Can't find thread corresponding to message")
	return MessageDb{
		ThreadID: t.Id,
		MessageID: messageID,
		Date: date,
		FromName: fromName,
		FromAddr: fromAddr,
	}
}

func newThreadLabelMapper(threadID int64, labelID int64) ThreadLabelMapper {
	return ThreadLabelMapper{
		ThreadID: threadID,
		LabelID:  labelID,
	}
}

func newThreadFlagMapper(threadID int64, flagID int64) ThreadFlagMapper {
	return ThreadFlagMapper{
		ThreadID: threadID,
		FlagID:   flagID,
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

	dbmap.AddTableWithName(ThreadDb{}, "thread").SetKeys(true, "Id").
		ColMap("ThreadID").SetUnique(true)

	dbmap.AddTableWithName(LabelDb{}, "label").SetKeys(true, "Id").
		ColMap("Label").SetUnique(true)

	dbmap.AddTableWithName(FlagDb{}, "flag").SetKeys(true, "Id").
		ColMap("Flag").SetUnique(true)

	dbmap.AddTableWithName(MessageDb{}, "message").SetKeys(true, "Id").
		ColMap("MessageID").SetUnique(true)

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

	sql = `create table if not exists thread_flag_mapper (
	thread_id integer, flag_id integer,
	foreign key(thread_id) references thread(id) on delete cascade,
	foreign key(flag_id) references flag(id) on delete cascade
	);
	create index thread_index on thread_flag_mapper(thread_id);
	create index flag_index on thread_flag_mapper(flag_id);
	`

	_, err = dbmap.Exec(sql)
	checkErr(err, "Unable to create thread_flag_mapper table")
	dbmap.AddTableWithName(ThreadFlagMapper{}, "thread_flag_mapper")

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
