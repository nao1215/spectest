package main

import (
	"errors"
	"net/http"
	"testing"

	"github.com/nao1215/spectest"
)

func TestGetUserSuccess(t *testing.T) {
	spectest.New().
		Mocks(getPreferencesMock, getUserMock).
		Handler(newApp().Router).
		Get("/user").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon", "is_contactable": true}`).
		End()
}

var getPreferencesMock = spectest.NewMock().
	Get("/preferences/12345").
	AddMatcher(func(r *http.Request, mr *spectest.MockRequest) error {
		// Custom matching func for URL Scheme
		if r.URL.Scheme != "http" {
			return errors.New("request did not have 'http' scheme")
		}
		return nil
	}).
	RespondWith().
	Body(`{"is_contactable": true}`).
	Status(http.StatusOK).
	End()

var getUserMock = spectest.NewMock().
	Get("/user/12345").
	RespondWith().
	Body(`{"name": "jon", "id": "1234"}`).
	Status(http.StatusOK).
	End()
