package main

import (
	"database/sql"
	"github.com/coopernurse/gorp"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func dbMain() {
	// initialize the DbMap
	dbmap := initDb()
	defer dbmap.Db.Close()

	// delete any existing rows
	err := dbmap.TruncateTables()
	checkErr(err, "TruncateTables failed")

	// create two posts
	p1 := newThread("938249", "foo")
	p2 := newThread("324985", "bar")

	// insert rows - auto increment PKs will be set properly after the insert
	err = dbmap.Insert(&p1, &p2)
	checkErr(err, "Insert failed")

	// use convenience SelectInt
	count, err := dbmap.SelectInt("select count(*) from thread")
	checkErr(err, "select count(*) failed")
	log.Println("Rows after inserting:", count)

	// update a row
	p2.ThreadID = "932849"
	count, err = dbmap.Update(&p2)
	checkErr(err, "Update failed")
	log.Println("Rows updated:", count)

	// fetch one row - note use of "post_id" instead of "Id" since column is aliased
	err = dbmap.SelectOne(&p2, "select * from thread where id=?", p2.Id)
	checkErr(err, "SelectOne failed")
	log.Println("p2 row:", p2)

	// fetch all rows
	var threads []ThreadDb
	_, err = dbmap.Select(&threads, "select * from thread order by id")
	checkErr(err, "Select failed")
	log.Println("All rows:")
	for x, p := range threads {
		log.Printf("    %d: %v\n", x, p)
	}

	// create relationships
	l1 := newLabel("git")
	l2 := newLabel("linux")
	err = dbmap.Insert(&l1, &l2)
	checkErr(err, "Insert failed")

	m1 := newThreadLabelMapper(p1.Id, l1.Id)
	m2 := newThreadLabelMapper(p1.Id, l2.Id)
	err = dbmap.Insert(&m1, &m2)
	checkErr(err, "Insert failed")

	// fetch all relationships
	var mappings []ThreadLabelMapper
	_, err = dbmap.Select(&mappings, "select * from thread_label_mapper")
	checkErr(err, "Select failed")
	log.Println("All mappings:")
	for x, p := range mappings {
		log.Printf("    %d: %v\n", x, p)
	}

	// delete row by PK
	count, err = dbmap.Delete(&p1)
	checkErr(err, "Delete failed")
	log.Println("Rows deleted:", count)

	// delete row manually via Exec
	_, err = dbmap.Exec("delete from thread where id=?", p2.Id)
	checkErr(err, "Exec failed")

	// confirm count is zero
	count, err = dbmap.SelectInt("select count(*) from thread")
	checkErr(err, "select count(*) failed")
	log.Println("Row count - should be zero:", count)

	log.Println("Done!")
}

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

type ThreadDb struct {
	Id       int64
	ThreadID string `db:"thread_id"`
	Subject  string
}

type LabelDb struct {
	Id      int64
	Label   string
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

func newThreadLabelMapper(threadID int64, labelID int64) ThreadLabelMapper {
	return ThreadLabelMapper{
		ThreadID: threadID,
		LabelID:  labelID,
	}
}

func initDb() *gorp.DbMap {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	db, err := sql.Open("sqlite3", "mail.db")
	checkErr(err, "sql.Open failed")

	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	// Enable foreign key support
	_, err = dbmap.Exec("pragma foreign_keys = ON")
	checkErr(err, "Failed to enable foreign key support")

	// add table for thread
	dbmap.AddTableWithName(ThreadDb{}, "thread").SetKeys(true, "Id").
		ColMap("ThreadId").SetUnique(true)

	// add table for label
	dbmap.AddTableWithName(LabelDb{}, "label").SetKeys(true, "Id").
		ColMap("Label").SetUnique(true)

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
