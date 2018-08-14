// Copyright 2018 Bernhard Reitinger. All rights reserved.

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

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// Reference is a spatial anchor which can be attached to a project or a project file.
//
// Currently this feature is not yet implemented fully.
type Reference struct {
	Key string `json:"key"`
}

// ProjectSimple is the basic structure representing a simple RexProject
type ProjectSimple struct {
	ID    string // auto-generated after getting the list of projects
	Name  string `json:"name"`
	Owner string `json:"owner"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
}

// ProjectSimpleList is a list ProjectSimple objects.
//
// Mainly required for JSON encoding/decoding
type ProjectSimpleList struct {
	Embedded struct {
		Projects []ProjectSimple `json:"projects"`
	} `json:"_embedded"`
}

// ProjectAddress defines the address information for a project
type ProjectAddress struct {
	AddressLine1 string `json:"addressLine1"`
	AddressLine2 string `json:"addressLine2"`
	AddressLine3 string `json:"addressLine3"`
	AddressLine4 string `json:"addressLine4"`
	PostCode     string `json:"postcode"`
	City         string `json:"city"`
	Region       string `json:"region"`
	Country      string `json:"country"`
}

// ProjectTransformation is used for the absoluteTransformation as well as for the relativeTransformation
// of a RexReference
type ProjectTransformation struct {
	Rotation struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	} `json:"rotation"`
	Position struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"position"`
}

var (
	apiProjects       = "/api/v2/projects"
	apiRexReferences  = "/api/v2/rexReferences"
	apiProjectByOwner = "/api/v2/projects/search/findAllByOwner?owner="
	apiProjectFiles   = "/api/v2/projectFiles/"
)

// String nicely prints a list of projects.
func (p ProjectSimpleList) String() string {
	var s string
	s += fmt.Sprintf("| %6s | %-20s | %-15s | %-65s |\n", "ID", "Name", "Owner", "Self Link")
	for _, proj := range p.Embedded.Projects {
		s += fmt.Sprintf("| %6s | %-20s | %-15s | %65s |\n", proj.ID, proj.Name, proj.Owner, proj.Links.Self.Href)
	}
	return s
}

// GetProjects gets all projects for the current user.
//
// This call only fetches the project list, but not the content of every project.
// Please use GetProject for getting the detailed project information.
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

// CreateProject creates a new project for the current user.
//
// The name is used as project name
func CreateProject(c *Client, name string, address *ProjectAddress, absoluteTransformation *ProjectTransformation) error {
	p := ProjectSimple{Name: name, Owner: c.User.UserID}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(p)

	req, _ := http.NewRequest("POST", RexBaseURL+apiProjects, b)
	req.Header.Add("accept", "application/json")
	c.token.SetAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 201 {
		return fmt.Errorf("Got server status %d with error: %s ", resp.StatusCode, body)
	}
	io.Copy(ioutil.Discard, resp.Body)

	projectSelfLink := gjson.Get(string(body), "_links.self.href").String()
	uuid := uuid.New().String()

	// Create a RexReference as well
	rexReference := struct {
		Project                string                 `json:"project"`
		RootReference          bool                   `json:"rootReference"`
		Key                    string                 `json:"key"`
		Address                *ProjectAddress        `json:"address"`
		AbsoluteTransformation *ProjectTransformation `json:"absoluteTransformation"`
	}{
		Project:                projectSelfLink,
		RootReference:          true,
		Key:                    uuid,
		Address:                address,
		AbsoluteTransformation: absoluteTransformation,
	}

	json.NewEncoder(b).Encode(rexReference)

	req, _ = http.NewRequest("POST", RexBaseURL+apiRexReferences, b)
	req.Header.Add("accept", "application/json")
	c.token.SetAuthHeader(req)

	resp, err = c.httpClient.Do(req)
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		return fmt.Errorf("Got server status %d with error: %s ", resp.StatusCode, body)
	}

	return nil
}

// UploadProjectFile uploads a new project file.
//
// The project is identified by the projectID (e.g. 1020). The file requires a name,
// which is displayed, but also a fileName which includes the suffix. The fileName is used
// for detecting the mimetype. The content of the file will be read from the io.Reader r.
func UploadProjectFile(c *Client, projectID string, name string, fileName string, r io.Reader) error {

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
