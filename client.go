// Copyright 2018 Bernhard Reitinger. All rights reserved.

package rex

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// Client stores all user relevant information. The client holds the
// authentication token but also the user information
// such as username, email, and some more.
//
// To create a new client you simply use the following code
//     client :=  NewClient(nil)
//     err := client.Login("<ClientId>", "<ClientSecret>")
type Client struct {
	User       *User        // Stores the user information
	Token      oauth2.Token // Contains the authentication token
	httpClient *http.Client // The actual net client
}

// Executor is an interface which is used to perform the actual
// REX request. This interface should be used for any REX API call.
// The Client structure is implementing this interface and performs the actual call.
type Executor interface {
	Execute(req *http.Request) (*http.Response, error)
}

var (
	apiAuth = "/oauth/token"
)

// Execute fullfills the Executor interface and performs a REX web request.
// Makes sure that the authentication token is set
func (c *Client) Execute(req *http.Request) (*http.Response, error) {

	req.Header.Add("accept", "application/json")
	c.Token.SetAuthHeader(req)
	return c.httpClient.Do(req)
}

// NewClient creates a new client instance
//
func NewClient(httpClient *http.Client) *Client {

	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	c := &Client{
		httpClient: httpClient,
	}
	return c
}

// NewClientWithToken takes token and reads the user information.
// If the token is not valid anymore, an error will be returned,
// else a new client will be provided.
func NewClientWithToken(token oauth2.Token, httpClient *http.Client) (*Client, error) {
	c := NewClient(httpClient)
	c.Token = token

	var err error
	c.User, err = GetCurrentUser(c)
	return c, err
}

// Login uses the user's authentication information and gets a new authentication token
func (c *Client) Login(clientID, clientSecret string) error {

	err := c.fetchToken(clientID, clientSecret)
	if err != nil {
		return err
	}

	c.User, err = GetCurrentUser(c)
	return err
}

func (c *Client) fetchToken(clientID, clientSecret string) error {

	req, _ := http.NewRequest("POST", RexBaseURL+apiAuth, strings.NewReader("grant_type=client_credentials"))

	token := clientID + ":" + clientSecret
	encodedToken := b64.StdEncoding.EncodeToString([]byte(token))
	req.Header.Add("authorization", "Basic "+encodedToken)
	req.Header.Add("content-type", "application/x-www-form-urlencoded; charset=ISO-8859-1")
	req.Header.Add("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Server did not respond properly")
	}

	return json.Unmarshal(body, &c.Token)
}
