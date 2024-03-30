package mocks_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nao1215/spectest/jsonpath/mocks"

	"github.com/nao1215/spectest"
)

func TestMocks(t *testing.T) {
	getUserMock := spectest.NewMock().
		Post("/user-api").
		AddMatcher(mocks.Equal("$.name", "jon")).
		AddMatcher(mocks.Equal("$.name", "jon")). // ensure body can be re read after running matcher
		RespondWith().
		Body(`{"name": "jon", "id": "1234"}`).
		Status(http.StatusOK).
		End()

	getPreferencesMock := spectest.NewMock().
		Get("/preferences-api").
		RespondWith().
		Body(`{"is_contactable": false}`).
		Status(http.StatusOK).
		End()

	spectest.New().
		Mocks(getUserMock, getPreferencesMock).
		Handler(myHandler()).
		Get("/user").
		Expect(t).
		Status(http.StatusOK).
		Body(`{"name": "jon", "is_contactable": false}`).
		End()
}

func myHandler() *http.ServeMux {
	handler := http.NewServeMux()
	handler.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		var user user
		if err := httpPost("/user-api", `{"name": "jon"}`, &user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var contactPreferences contactPreferences
		if err := httpGet("/preferences-api", &contactPreferences); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := userResponse{
			Name:          user.Name,
			IsContactable: contactPreferences.IsContactable,
		}

		bytes, _ := json.Marshal(response)
		_, err := w.Write(bytes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
	return handler
}

type user struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type contactPreferences struct {
	IsContactable bool `json:"is_contactable"`
}

type userResponse struct {
	Name          string `json:"name"`
	IsContactable bool   `json:"is_contactable"`
}

func httpGet(path string, response interface{}) error {
	res, err := http.DefaultClient.Get(fmt.Sprintf("http://localhost:8080%s", path))
	if err != nil {
		return err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, response)
	if err != nil {
		return err
	}

	return nil
}

func httpPost(path string, requestBody string, response interface{}) error {
	res, err := http.DefaultClient.Post(fmt.Sprintf("http://localhost:8080%s", path), "application/json", strings.NewReader(requestBody))
	if err != nil {
		return err
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, response)
	if err != nil {
		return err
	}

	return nil
}
