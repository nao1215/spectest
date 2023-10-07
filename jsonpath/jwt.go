package jsonpath

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-spectest/spectest/jsonpath/jsonpath"
)

const (
	jwtHeaderIndex  = 0
	jwtPayloadIndex = 1
)

// JWTHeaderEqual asserts that the JWT header matches the expected value
func JWTHeaderEqual(tokenSelector func(*http.Response) (string, error), expression string, expected interface{}) func(*http.Response, *http.Request) error {
	return jwtEqual(tokenSelector, expression, expected, jwtHeaderIndex)
}

// JWTPayloadEqual asserts that the JWT payload matches the expected value
func JWTPayloadEqual(tokenSelector func(*http.Response) (string, error), expression string, expected interface{}) func(*http.Response, *http.Request) error {
	return jwtEqual(tokenSelector, expression, expected, jwtPayloadIndex)
}

func jwtEqual(tokenSelector func(*http.Response) (string, error), expression string, expected interface{}, index int) func(*http.Response, *http.Request) error {
	return func(response *http.Response, request *http.Request) error {
		token, err := tokenSelector(response)
		if err != nil {
			return err
		}

		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			splitErr := errors.New("invalid token: token should contain header, payload and secret")
			return splitErr
		}

		decodedPayload, PayloadErr := base64Decode(parts[index])
		if PayloadErr != nil {
			return fmt.Errorf("invalid jwt: %s", PayloadErr.Error())
		}

		value, err := jsonpath.JSONPath(bytes.NewReader(decodedPayload), expression)
		if err != nil {
			return err
		}

		if !jsonpath.ObjectsAreEqual(value, expected) {
			return fmt.Errorf("\"%s\" not equal to \"%s\"", value, expected)
		}

		return nil
	}
}

func base64Decode(src string) ([]byte, error) {
	if l := len(src) % 4; l > 0 {
		src += strings.Repeat("=", 4-l)
	}

	decoded, err := base64.URLEncoding.DecodeString(src)
	if err != nil {
		errMsg := fmt.Errorf("decoding Error %s", err)
		return nil, errMsg
	}
	return decoded, nil
}
