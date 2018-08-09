package rex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// Reference is the RexReference for a simple REX item
type Reference struct {
	Key string `json:"key"`
}

// ProjectSimple is the basic structure for storing REX content
type ProjectSimple struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
}

// Projects is a list of projects
type Projects struct {
	Embedded struct {
		Projects []ProjectSimple `json:"projects"`
	} `json:"_embedded"`
}

var (
	apiProjects = "/api/v2/projects/search/findAllByOwner?owner=user-id-m"
)

func (p ProjectSimple) String() string {
	return fmt.Sprintf("| %-20s | %-15s | %s", p.Name, p.Owner, p.Links.Self.Href)
}

// GetProjects gets all projects for the current user
func GetProjects(c *Client) {
	req, _ := http.NewRequest("GET", RexBaseURL+apiProjects, nil)
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

	var projects Projects
	err = json.Unmarshal(body, &projects)

	if err != nil {
		panic(err)
	}

	for _, p := range projects.Embedded.Projects {
		fmt.Println(p)
	}

	// fmt.Println(string(body))
}

// TODO test code
func UpdateProjectFile(c *Client) {
	var body = []byte(`{"type": "rex"}`)
	req, _ := http.NewRequest("PATCH", RexBaseURL+"/api/v2/projectFiles/1044", bytes.NewBuffer(body))
	c.token.SetAuthHeader(req)
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Status)
}

// TODO test code
func UploadFileContent(c *Client, filePath string) {

	file, _ := os.Open(filePath)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.Close()

	req, _ := http.NewRequest("POST", RexBaseURL+"/api/v2/projectFiles/1044/file", body)
	c.token.SetAuthHeader(req)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()
}
