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

// Reference is currently a simple struct which only contains the key
//
// However the Reference (RexReference) will be used to permanently attach location information to any project or
// project item
type Reference struct {
	Key string `json:"key"`
}

// ProjectSimple is the basic structure representing a simple RexProject
type ProjectSimple struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
}

// ProjectSimpleList is a list of projects
type ProjectSimpleList struct {
	Embedded struct {
		Projects []ProjectSimple `json:"projects"`
	} `json:"_embedded"`
}

var (
	apiProjects       = "/api/v2/projects"
	apiProjectByOwner = "/api/v2/projects/search/findAllByOwner?owner="
)

// String nicely prints a list of projects
func (p ProjectSimpleList) String() string {
	var s string
	for _, proj := range p.Embedded.Projects {
		s += fmt.Sprintf("| %-20s | %-15s | %65s |\n", proj.Name, proj.Owner, proj.Links.Self.Href)
	}
	return s
}

// GetProjects gets all projects for the user current user
func GetProjects(c *Client) (*ProjectSimpleList, error) {
	req, _ := http.NewRequest("GET", RexBaseURL+apiProjectByOwner+c.User.UserID, nil)
	c.token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	var projects ProjectSimpleList
	err = json.NewDecoder(resp.Body).Decode(&projects)
	return &projects, err
}

// CreateProject creates a new project for the current user
func CreateProject(c *Client, name string) error {
	p := ProjectSimple{Name: name, Owner: c.User.UserID}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(p)

	req, _ := http.NewRequest("POST", RexBaseURL+apiProjects, b)
	c.token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 201 {
		return fmt.Errorf("Got server status %d with error: %s ", resp.StatusCode, body)
	}
	return nil
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
