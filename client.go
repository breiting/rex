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

// Client is the main client for accessing the REX API
type Client struct {
	ClientID     string
	ClientSecret string
	User         *User // stores the current user

	httpClient *http.Client
	token      oauth2.Token
}

var (
	apiAuth = "/oauth/token"
)

func (c *Client) fetchToken() error {

	req, _ := http.NewRequest("POST", RexBaseURL+apiAuth, strings.NewReader("grant_type=client_credentials"))

	token := c.ClientID + ":" + c.ClientSecret
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

	return json.Unmarshal(body, &c.token)
}

// NewClient creates a new client instance and authenticates the user with the given credentials
//
// This call also fetches the user information, so that the API user already has the User information
// in the pocket. E.g. the self link is required for performing sub-sequent operations. This can be
// retrieved by
//     User.SelfLink.
//
func NewClient(clientID string, clientSecret string, httpClient *http.Client) (*Client, error) {

	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	c := &Client{
		httpClient:   httpClient,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	err := c.fetchToken()
	if err != nil {
		return nil, err
	}

	c.User, err = GetCurrentUser(c)
	return c, err
}
