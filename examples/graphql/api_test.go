package main_test

import (
	"net/http"
	"testing"

	"github.com/go-spectest/spectest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"github.com/steinfletcher/apitest/examples/graphql/graph"
)

func TestQueryEmpty(t *testing.T) {
	spectest.New().
		Handler(graph.NewHandler()).
		Post("/query").
		GraphQLQuery(`query {
			todos {
				text
				done
				user {
					name
				}
			}
		}`).
		Expect(t).
		Status(http.StatusOK).
		Body(`{
		  "data": {
			"todos": []
		  }
		}`).
		End()
}

func TestQueryWithTodo(t *testing.T) {
	handler := graph.NewHandler()

	spectest.New().
		Handler(handler).
		Post("/query").
		JSON(`{"query": "mutation { createTodo(input:{text:\"todo\", userId:\"4\"}) { user { id } text done } }"}`).
		Expect(t).
		Status(http.StatusOK).
		Assert(jsonpath.Equal("$.data.createTodo.user.id", "4")).
		End()

	spectest.New().
		Handler(handler).
		Post("/query").
		GraphQLQuery("query { todos { text done user { name } } }").
		Expect(t).
		Status(http.StatusOK).
		Body(`{
		  "data": {
			"todos": [
			  {
				"text": "todo",
				"done": false,
				"user": {
				  "name": "user"
				}
			  }
			]
		  }
		}`).
		End()
}
