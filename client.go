package rex

import (
	b64 "encoding/base64"
	"encoding/json"
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

	httpClient *http.Client
	token      oauth2.Token
}

var (
	apiAuth = "/oauth/token"
)

func (c *Client) fetchToken() {

	req, _ := http.NewRequest("POST", RexBaseURL+apiAuth, strings.NewReader("grant_type=client_credentials"))

	token := c.ClientID + ":" + c.ClientSecret
	encodedToken := b64.StdEncoding.EncodeToString([]byte(token))
	req.Header.Add("authorization", "Basic "+encodedToken)
	req.Header.Add("content-type", "application/x-www-form-urlencoded; charset=ISO-8859-1")
	req.Header.Add("accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		panic(resp.Status)
	}

	err = json.Unmarshal(body, &c.token)
	if err != nil {
		panic(err)
	}
}

// NewClient creates a new client instance and authenticates the user with the given credentials
func NewClient(clientID string, clientSecret string, httpClient *http.Client) *Client {

	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	c := &Client{
		httpClient:   httpClient,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	c.fetchToken()
	return c
}
