package main

import (
	"net/http"
	"testing"

	"github.com/nao1215/spectest"
	"github.com/nao1215/spectest/jsonpath"
)

func TestGetUserCookieMatching(t *testing.T) {
	spectest.New().
		Handler(newRouter()).
		Get("/user/1234").
		Expect(t).
		Cookies(spectest.NewCookie("CookieForAndy").Value("Andy")).
		Status(http.StatusOK).
		End()
}

func TestGetUserSuccess(t *testing.T) {
	spectest.New().
		Handler(newRouter()).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestGetUserSuccessJSONPath(t *testing.T) {
	spectest.New().
		Handler(newRouter()).
		Get("/user/1234").
		Expect(t).
		Assert(jsonpath.Equal(`$.id`, "1234")).
		Status(http.StatusOK).
		End()
}

func TestGetUserNotFound(t *testing.T) {
	spectest.New().
		Handler(newRouter()).
		Get("/user/1515").
		Expect(t).
		Status(http.StatusNotFound).
		End()
}
