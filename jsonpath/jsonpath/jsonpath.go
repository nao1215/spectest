// Package jsonpath is not referenced by user code.
package jsonpath

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

// Contains is a convenience function to assert that a jsonpath expression extracts a value in an array
func Contains(expression string, expected interface{}, data io.Reader) error {
	value, err := JSONPath(data, expression)
	if err != nil {
		return err
	}
	ok, found := IncludesElement(value, expected)
	if !ok {
		return fmt.Errorf("\"%s\" could not be applied builtin len()", expected)
	}
	if !found {
		return fmt.Errorf("\"%s\" does not contain \"%s\"", value, expected)
	}
	return nil
}

// Equal is a convenience function to assert that a jsonpath expression extracts a value
func Equal(expression string, expected interface{}, data io.Reader) error {
	value, err := JSONPath(data, expression)
	if err != nil {
		return err
	}
	if !ObjectsAreEqual(value, expected) {
		return fmt.Errorf("\"%s\" not equal to \"%s\"", value, expected)
	}
	return nil
}

// NotEqual is a function to check json path expression value is not equal to given value
func NotEqual(expression string, expected interface{}, data io.Reader) error {
	value, err := JSONPath(data, expression)
	if err != nil {
		return err
	}

	if ObjectsAreEqual(value, expected) {
		return fmt.Errorf("\"%s\" value is equal to \"%s\"", expression, expected)
	}
	return nil
}

// Length asserts that value is the expected length, determined by reflect.Len
func Length(expression string, expectedLength int, data io.Reader) error {
	value, err := JSONPath(data, expression)
	if err != nil {
		return err
	}

	if value == nil {
		return errors.New("value is null")
	}

	v := reflect.ValueOf(value)
	if v.Len() != expectedLength {
		return fmt.Errorf("\"%d\" not equal to \"%d\"", v.Len(), expectedLength)
	}
	return nil
}

// GreaterThan asserts that value is greater than the given length, determined by reflect.Len
func GreaterThan(expression string, minimumLength int, data io.Reader) error {
	value, err := JSONPath(data, expression)
	if err != nil {
		return err
	}

	if value == nil {
		return fmt.Errorf("value is null")
	}

	v := reflect.ValueOf(value)
	if v.Len() < minimumLength {
		return fmt.Errorf("\"%d\" is greater than \"%d\"", v.Len(), minimumLength)
	}
	return nil
}

// LessThan asserts that value is less than the given length, determined by reflect.Len
func LessThan(expression string, maximumLength int, data io.Reader) error {
	value, err := JSONPath(data, expression)
	if err != nil {
		return err
	}

	if value == nil {
		return fmt.Errorf("value is null")
	}

	v := reflect.ValueOf(value)
	if v.Len() > maximumLength {
		return fmt.Errorf("\"%d\" is less than \"%d\"", v.Len(), maximumLength)
	}
	return nil
}

// Present asserts that value returned by the expression is present
func Present(expression string, data io.Reader) error {
	value, _ := JSONPath(data, expression)
	if isEmpty(value) {
		return fmt.Errorf("value not present for expression: '%s'", expression)
	}
	return nil
}

// NotPresent asserts that value returned by the expression is not present
func NotPresent(expression string, data io.Reader) error {
	value, _ := JSONPath(data, expression)
	if !isEmpty(value) {
		return fmt.Errorf("value present for expression: '%s'", expression)
	}
	return nil
}

// JSONPath evaluates the given expression against the given JSON document
func JSONPath(reader io.Reader, expression string) (interface{}, error) {
	v := interface{}(nil)
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &v)
	if err != nil {
		return nil, err
	}

	value, err := jsonpath.Get(expression, v)
	if err != nil {
		return nil, fmt.Errorf("evaluating '%s' resulted in error: '%s'", expression, err)
	}
	return value, nil
}

// IncludesElement courtesy of github.com/stretchr/testify
func IncludesElement(list interface{}, element interface{}) (ok, found bool) {
	listValue := reflect.ValueOf(list)
	elementValue := reflect.ValueOf(element)
	defer func() {
		if e := recover(); e != nil {
			ok = false
			found = false
		}
	}()

	if reflect.TypeOf(list).Kind() == reflect.String {
		return true, strings.Contains(listValue.String(), elementValue.String())
	}

	if reflect.TypeOf(list).Kind() == reflect.Map {
		mapKeys := listValue.MapKeys()
		for i := 0; i < len(mapKeys); i++ {
			if ObjectsAreEqual(mapKeys[i].Interface(), element) {
				return true, true
			}
		}
		return true, false
	}

	for i := 0; i < listValue.Len(); i++ {
		if ObjectsAreEqual(listValue.Index(i).Interface(), element) {
			return true, true
		}
	}
	return true, false
}

// ObjectsAreEqual return expected and acutal are equal or not
func ObjectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

func isEmpty(object interface{}) bool {
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}
