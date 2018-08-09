package rex

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// User is the basic structure for the current user information
type User struct {
	UserID    string   `json:"userId"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Roles     []string `json:"roles"`
	Links     struct {
		User struct {
			Href string `json:"href"`
		} `json:"user"`
	} `json:"_links"`
}

var (
	apiCurrentUser = "/api/v2/users/current"
)

func (u User) String() string {
	s := fmt.Sprintf("| UserId   | %-65s |\n", u.UserID)
	s += fmt.Sprintf("| Username | %-65s |\n", u.Username)
	s += fmt.Sprintf("| Firstname| %-65s |\n", u.FirstName)
	s += fmt.Sprintf("| Lastname | %-65s |\n", u.LastName)
	s += fmt.Sprintf("| Email    | %-65s |\n", u.Email)
	s += fmt.Sprintf("| Link     | %-65s |\n", u.Links.User.Href)

	return s
}

// GetCurrentUser gets the user details of the current user
func GetCurrentUser(c *Client) User {
	req, _ := http.NewRequest("GET", RexBaseURL+apiCurrentUser, nil)
	c.token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var u User
	err = json.Unmarshal(body, &u)

	if err != nil {
		panic(err)
	}
	return u
}
