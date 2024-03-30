package main

import (
	"net/http"
	"testing"

	"github.com/nao1215/spectest"
)

func TestGetUserWithDefaultReportFormatter(t *testing.T) {
	spectest.New("gets the user 1").
		Report(spectest.SequenceDiagram()).
		CustomHost("user-service").
		Mocks(getPreferencesMock, getUserMock).
		Handler(newApp().Router).
		Post("/user/search").
		JSON(`{"name":"jan"}`).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(`{"name": "jon", "is_contactable": true}`).
		End()
}

func TestGetUserWithDefaultReportFormatterOverridingPath(t *testing.T) {
	spectest.New("gets the user 2").
		CustomHost("user-service").
		Report(spectest.SequenceDiagram(".sequence-diagrams")).
		Mocks(getPreferencesMock, getUserMock).
		Handler(newApp().Router).
		Post("/user/search").
		JSON(`{"name":"jan"}`).
		Expect(t).
		Status(http.StatusOK).
		Header("Content-Type", "application/json").
		Body(`{"name": "jon", "is_contactable": true}`).
		End()
}

var getPreferencesMock = spectest.NewMock().
	Get("http://preferences/api/preferences/12345").
	RespondWith().
	Body(`{"is_contactable": true}`).
	Status(http.StatusOK).
	End()

var getUserMock = spectest.NewMock().
	Get("http://users/api/user/12345").
	RespondWith().
	Body(`{"name": "jon", "id": "1234"}`).
	Status(http.StatusOK).
	End()
