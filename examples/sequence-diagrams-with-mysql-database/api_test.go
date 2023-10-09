package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/go-spectest/spectest"
	"github.com/go-spectest/spectest/x/db"
	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"
)

// This test requires a mysql database to run

var recorder *spectest.Recorder

func init() {
	recorder = spectest.NewTestRecorder()

	// Wrap your database driver of choice with a recorder
	// and register it so you can use it later
	wrappedDriver := db.WrapWithRecorder("mysql", recorder)
	sql.Register("wrappedMysql", wrappedDriver)
}

func TestGetUserWithDefaultReportFormatter(t *testing.T) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	defer recorder.Reset()
	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
		db.MustExec(q, username, true)
	})

	apiTest("gets the user").
		Debug().
		Mocks(getUserMock(username)).
		Get("/user").
		Query("name", username).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
		End()
}

func TestPostUserWithDefaultReportFormatter(t *testing.T) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.SkipNow()
	}

	defer recorder.Reset()
	username := uuid.NewV4().String()[0:7]

	DBSetup(dsn, func(db *sqlx.DB) {
		q := "INSERT INTO users (username, is_contactable) VALUES (?, ?)"
		db.MustExec(q, username, true)
	})

	apiTest("creates a user").
		Debug().
		Mocks(postUserMock(username)).
		Post("/user").
		Body(fmt.Sprintf(`{"name": "%s", "is_contactable": true}`, username)).
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
	dsn := os.Getenv("MYSQL_DSN")

	// Connect using the previously registered driver
	testDB, err := sqlx.Connect("wrappedMysql", dsn)
	if err != nil {
		panic(err)
	}

	// You can also use the WrapConnectorWithRecorder method
	// without having to register a new database driver
	//
	// cfg, err := mysql.ParseDSN(dsn)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// connector, err := mysql.NewConnector(cfg)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// wrappedConnector := apitestdb.WrapConnectorWithRecorder(connector, "mysql", recorder)
	// testDB := sqlx.NewDb(sql.OpenDB(wrappedConnector), "mysql")

	app := newApp(testDB)

	return spectest.New(name).
		Recorder(recorder).
		Report(spectest.SequenceDiagram()).
		Handler(app.Router)
}
