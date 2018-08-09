package rex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"regexp"

	"github.com/tidwall/gjson"
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
	ID    string // only set by the client for convenience
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
	apiProjectFiles   = "/api/v2/projectFiles/"
)

// String nicely prints a list of projects
func (p ProjectSimpleList) String() string {
	var s string
	s += fmt.Sprintf("| %6s | %-20s | %-15s | %-65s |\n", "ID", "Name", "Owner", "Self Link")
	for _, proj := range p.Embedded.Projects {
		s += fmt.Sprintf("| %6s | %-20s | %-15s | %65s |\n", proj.ID, proj.Name, proj.Owner, proj.Links.Self.Href)
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

	// set ID for convenience
	for i, p := range projects.Embedded.Projects {
		re, _ := regexp.Compile("/projects/(.*)")
		values := re.FindStringSubmatch(p.Links.Self.Href)
		if len(values) > 0 {
			projects.Embedded.Projects[i].ID = values[1]
		}
	}
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

// UploadProjectFiles uploads a list of local files to a given project
func UploadProjectFiles(c *Client, projectID string, name string, fileName string, r io.Reader) error {

	projectFile := struct {
		Name    string `json:"name"`
		Project string `json:"project"`
	}{
		Name:    name,
		Project: RexBaseURL + apiProjects + "/" + projectID,
	}

	// Create project file
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(projectFile)

	req, _ := http.NewRequest("POST", RexBaseURL+apiProjectFiles, b)
	req.Header.Add("accept", "application/json")
	c.token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println(string(body))

	if resp.StatusCode != 201 {
		return fmt.Errorf("Got server status %d with error: %s ", resp.StatusCode, body)
	}

	// Upload the actual payload
	uploadURL := gjson.Get(string(body), "_links.file\\.upload.href").String()
	io.Copy(ioutil.Discard, resp.Body)
	return uploadFileContent(c, uploadURL, fileName, r)
}

func uploadFileContent(c *Client, uploadURL string, fileName string, r io.Reader) error {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fileName)
	io.Copy(part, r)
	writer.Close()

	req, _ := http.NewRequest("POST", uploadURL, body)
	c.token.SetAuthHeader(req)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	io.Copy(ioutil.Discard, resp.Body)
	return err
}

// UpdateProjectFile - test code
func updateProjectFile(c *Client) {
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
