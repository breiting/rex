// Copyright 2018 Bernhard Reitinger. All rights reserved.

package rex

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/tidwall/gjson"
)

// User stores information of the current user.
//
// The user can either contain the currentUser information,
// but also information from a user query. The SelfLink can be
// used to directly access the data, but is also often required
// for other operations (e.g. insert a project).
type User struct {
	UserID    string `json:"userId"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	LastLogin string `json:"lastLogin,omitempty"`
	SelfLink  string
	Roles     []string `json:"roles,omitempty"`
	Links     struct {
		User struct {
			Href string `json:"href"`
		} `json:"user"`
	} `json:"_links,omitempty"`
}

var (
	apiCurrentUser = "/api/v2/users/current"
	apiUsers       = "/api/v2/users"
	apiFindByEmail = "/api/v2/users/search/findUserIdByEmail?email="
	apiFindByID    = "/api/v2/users/search/findByUserId?userId="
)

// String nicely prints out the user information.
func (u User) String() string {
	s := fmt.Sprintf("|-------------------------------------------------------------------------------|\n")
	s += fmt.Sprintf("| UserId    | %-65s |\n", u.UserID)
	s += fmt.Sprintf("| Username  | %-65s |\n", u.Username)
	s += fmt.Sprintf("| Firstname | %-65s |\n", u.FirstName)
	s += fmt.Sprintf("| Lastname  | %-65s |\n", u.LastName)
	s += fmt.Sprintf("| Email     | %-65s |\n", u.Email)
	s += fmt.Sprintf("| LastLogin | %-65s |\n", u.LastLogin)
	s += fmt.Sprintf("| Self      | %-65s |\n", u.SelfLink)
	s += fmt.Sprintf("|-------------------------------------------------------------------------------|\n")

	return s
}

// GetCurrentUser gets the user details of the current user.
//
// The current user is the one which has been identified by the authentication token.
// When a new client is created by NewClient, this function will already be called implicitly.
func GetCurrentUser(c *Client) (*User, error) {
	req, _ := http.NewRequest("GET", RexBaseURL+apiCurrentUser, nil)
	c.token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var u User
	err = json.Unmarshal(body, &u)
	u.SelfLink = u.Links.User.Href // assign self link
	return &u, err
}

// GetTotalNumberOfUsers returns the number of registered users.
//
// Requires admin permissions!
func GetTotalNumberOfUsers(c *Client) (uint64, error) {
	req, _ := http.NewRequest("GET", RexBaseURL+apiUsers, nil)
	c.token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	return gjson.Get(string(body), "page.totalElements").Uint(), nil
}

// GetUserByEmail retrieves the user information based on a given email address
func GetUserByEmail(c *Client, email string) (*User, error) {

	req, _ := http.NewRequest("GET", RexBaseURL+apiFindByEmail+email, nil)
	c.token.SetAuthHeader(req)

	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// check if the user can be found
	var user User
	err = json.NewDecoder(r.Body).Decode(&user)
	io.Copy(ioutil.Discard, r.Body)

	if err != nil || user.UserID == "" {
		return &User{}, fmt.Errorf("user not found")
	}

	// Fetch actual user information based on the retrieved UserID
	req, _ = http.NewRequest("GET", RexBaseURL+apiFindByID+user.UserID, nil)
	c.token.SetAuthHeader(req)
	r, err = c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, r.Body)
	}()

	err = json.NewDecoder(r.Body).Decode(&user)
	return &user, nil
}
