package main

import "testing"

func TestDb(t *testing.T) {
	// initialize the DbMap
	dbmap := initDb(true)
	defer dbmap.Db.Close()

	// delete any existing rows
	err := dbmap.TruncateTables()
	checkTErr(t, err, "TruncateTables failed")

	// create two posts
	p1 := newThread("938249", "foo")
	p2 := newThread("324985", "bar")

	// insert rows - auto increment PKs will be set properly after the insert
	err = dbmap.Insert(&p1, &p2)
	checkTErr(t, err, "Insert failed")

	// use convenience SelectInt
	count, err := dbmap.SelectInt("select count(*) from thread")
	checkTErr(t, err, "select count(*) failed")
	t.Log("Rows after inserting:", count)

	// update a row
	p2.ThreadID = "932849"
	count, err = dbmap.Update(&p2)
	checkTErr(t, err, "Update failed")
	t.Log("Rows updated:", count)

	// fetch one row - note use of "post_id" instead of "Id" since column is aliased
	err = dbmap.SelectOne(&p2, "select * from thread where id=?", p2.Id)
	checkTErr(t, err, "SelectOne failed")
	t.Log("p2 row:", p2)

	// fetch all rows
	var threads []ThreadDb
	_, err = dbmap.Select(&threads, "select * from thread order by id")
	checkTErr(t, err, "Select failed")
	t.Log("All rows:")
	for x, p := range threads {
		t.Logf("    %d: %v\n", x, p)
	}

	// create relationships
	l1 := newLabel("git")
	l2 := newLabel("linux")
	err = dbmap.Insert(&l1, &l2)
	checkTErr(t, err, "Insert failed")

	m1 := newThreadLabelMapper(p1.Id, l1.Id)
	m2 := newThreadLabelMapper(p1.Id, l2.Id)
	err = dbmap.Insert(&m1, &m2)
	checkTErr(t, err, "Insert failed")

	// fetch all relationships
	var mappings []ThreadLabelMapper
	_, err = dbmap.Select(&mappings, "select * from thread_label_mapper")
	checkTErr(t, err, "Select failed")
	t.Log("All mappings:")
	for x, p := range mappings {
		t.Logf("    %d: %v\n", x, p)
	}

	// delete row by PK
	count, err = dbmap.Delete(&p1)
	checkTErr(t, err, "Delete failed")
	t.Log("Rows deleted:", count)

	// delete row manually via Exec
	_, err = dbmap.Exec("delete from thread where id=?", p2.Id)
	checkTErr(t, err, "Exec failed")

	// confirm count is zero
	count, err = dbmap.SelectInt("select count(*) from thread")
	checkTErr(t, err, "select count(*) failed")
	t.Log("Row count - should be zero:", count)

	t.Log("Done!")
}

func checkTErr(t *testing.T, err error, msg string) {
	if err != nil {
		t.Error(msg, err)
	}
}
