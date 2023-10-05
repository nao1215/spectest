package main

import (
	"net/http"
	"testing"

	apitest "github.com/go-spectest/spectest"
)

func TestGetUserCookieMatching(t *testing.T) {
	apitest.New().
		Handler(newApp().Router).
		Get("/user/1234").
		Expect(t).
		Cookies(apitest.NewCookie("TomsFavouriteDrink").
			Value("Beer").
			Path("/")).
		Status(http.StatusOK).
		End()
}

func TestGetUserSuccess(t *testing.T) {
	apitest.New().
		Handler(newApp().Router).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestGetUserSuccessJSONPath(t *testing.T) {
	apitest.New().
		Handler(newApp().Router).
		Get("/user/1234").
		Expect(t).
		Assert(jsonpath.Equal(`$.id`, "1234")).
		Status(http.StatusOK).
		End()
}

func TestGetUserNotFound(t *testing.T) {
	apitest.New().
		Handler(newApp().Router).
		Get("/user/1515").
		Expect(t).
		Status(http.StatusNotFound).
		End()
}
