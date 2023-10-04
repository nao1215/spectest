package main

import (
	"net/http"
	"testing"

	apitest "github.com/nao1215/spectest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
)

func TestGetUser_CookieMatching(t *testing.T) {
	apitest.New().
		Handler(newRouter()).
		Get("/user/1234").
		Expect(t).
		Cookies(apitest.NewCookie("CookieForAndy").Value("Andy")).
		Status(http.StatusOK).
		End()
}

func TestGetUser_Success(t *testing.T) {
	apitest.New().
		Handler(newRouter()).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestGetUser_Success_JSONPath(t *testing.T) {
	apitest.New().
		Handler(newRouter()).
		Get("/user/1234").
		Expect(t).
		Assert(jsonpath.Equal(`$.id`, "1234")).
		Status(http.StatusOK).
		End()
}

func TestGetUser_NotFound(t *testing.T) {
	apitest.New().
		Handler(newRouter()).
		Get("/user/1515").
		Expect(t).
		Status(http.StatusNotFound).
		End()
}
