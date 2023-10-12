package spectest

import "fmt"

// basicAuth is represents the basic auth credentials
type basicAuth struct {
	// userName is the userName for basic auth
	userName string
	// password is the password for basic auth
	password string
}

// newBasicAuth creates a new basic auth
func newBasicAuth(userName, password string) basicAuth {
	return basicAuth{
		userName: userName,
		password: password,
	}
}

// isUserNameEmpty returns true if the userName is empty
func (b basicAuth) isUserNameEmpty() bool {
	return b.userName == ""
}

// isPasswordEmpty returns true if the password is empty
func (b basicAuth) isPasswordEmpty() bool {
	return b.password == ""
}

// auth will authenticate the user
// auth will return an error if the user is not authenticated
func (b basicAuth) auth(userName, password string) error {
	if b.userName != userName {
		return fmt.Errorf("basic auth request username '%s' did not match mock username '%s'",
			userName, b.userName)
	}

	if b.password != password {
		return fmt.Errorf("basic auth request password '%s' did not match mock password '%s'",
			password, b.password)
	}
	return nil
}
