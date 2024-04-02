package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/nao1215/spectest"
	"github.com/nao1215/spectest/x/db"
	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"
)

var recorder *spectest.Recorder

func init() {
	recorder = spectest.NewTestRecorder()

	// Wrap your database driver of choice with a recorder
	// and register it so you can use it later
	wrappedDriver := db.WrapWithRecorder("sqlite3", recorder)
	sql.Register("wrappedSqlite", wrappedDriver)
}

// TODO: fix below code
/*
func TestGetUserWithDefaultReportFormatter(t *testing.T) {
	t.Setenv("SQLITE_DSN", "./foo.db")

	username := uuid.NewV4().String()[0:7]

	DBSetup("./foo.db", func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
		db.MustExec(q, username, true)
	})

	spectest.New("gets the user").
		Mocks(getUserMock(username)).
		Get("/some-really-long-path-so-we-can-observe-truncation-here-whey").
		Query("name", username).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
		End()
}
*/

func TestPostUserWithDefaultReportFormatter(t *testing.T) {
	dsn := os.Getenv("SQLITE_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
		db.MustExec(q, username, true)
	})

	spectest.New("creates a user").
		Mocks(postUserMock(username)).
		Post("/user").
		JSON(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		End()
}

func getUserMock(username string) *spectest.Mock {
	return spectest.NewMock().
		Get("http://users/api/user").
		Query("id", username).
		RespondWith().
		Body(fmt.Sprintf(`{"name": "%s", "id": "1234"}`, username)).
		Status(http.StatusOK).
		End()
}

func postUserMock(username string) *spectest.Mock {
	return spectest.NewMock().
		Post("http://users/api/user").
		Body(fmt.Sprintf(`{"name": "%s"}`, username)).
		RespondWith().
		Status(http.StatusOK).
		End()
}

func apiTest(name string) *spectest.SpecTest {
	dsn := os.Getenv("SQLITE_DSN")

	// Connect using the previously registered driver
	testDB, err := sqlx.Connect("wrappedSqlite", dsn)
	if err != nil {
		panic(err)
	}

	app := newApp(testDB)

	return spectest.New(name).
		Recorder(recorder).
		Report(spectest.SequenceDiagram()).
		Handler(app.Router)
}
