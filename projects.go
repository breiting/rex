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
type Reference struct {
	Key             string                 `json:"key"`
	Project         string                 `json:"project"`
	ParentReference string                 `json:"parentReference"`
	RootReference   bool                   `json:"rootReference"`
	Address         *ProjectAddress        `json:"address"`
	AbsTransform    *ProjectTransformation `json:"absoluteTransformation"`
	RelTransform    *ProjectTransformation `json:"relativeTransformation"`
	FileTransform   *FileTransformation    `json:"fileTransformation"`
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

// FileTransformation is used for defining the relationship between the RexReference and the actual file.
type FileTransformation struct {
	Rotation struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		Z float64 `json:"z"`
	} `json:"rotation"`
	Position struct {
		Type        string    `json:"type"`
		Coordinates []float64 `json:"coordinates"`
	} `json:"position"`
	Scale float64 `json:"scale"`
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
func GetProjects(e Executor, userID string) (*ProjectSimpleList, error) {
	req, _ := http.NewRequest("GET", RexBaseURL+apiProjectByOwner+userID, nil)

	resp, err := e.Execute(req)
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

// Creates a new RexReference using the REX API
func createRexReference(e Executor, r *Reference) (string, error) {

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(r)

	req, _ := http.NewRequest("POST", RexBaseURL+apiRexReferences, b)
	resp, err := e.Execute(req)
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()
	if err != nil {
		return "", err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		return "", fmt.Errorf("Got server status %d with error: %s ", resp.StatusCode, body)
	}
	return gjson.Get(string(body), "_links.self.href").String(), nil
}

// CreateProject creates a new project for the current user.
//
// The name is used as project name
func CreateProject(e Executor, userID, name string, address *ProjectAddress, absoluteTransformation *ProjectTransformation) error {
	p := ProjectSimple{Name: name, Owner: userID}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(p)

	req, _ := http.NewRequest("POST", RexBaseURL+apiProjects, b)
	resp, err := e.Execute(req)
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
	rexReference := Reference{
		Project:       projectSelfLink,
		RootReference: true,
		Key:           uuid,
		Address:       address,
		AbsTransform:  absoluteTransformation,
	}

	_, err = createRexReference(e, &rexReference)
	return err
}

// UploadProjectFile uploads a new project file.
//
// The project is identified by the projectID (e.g. 1020). The file requires a name,
// which is displayed, but also a fileName which includes the suffix. The fileName is used
// for detecting the mimetype. The content of the file will be read from the io.Reader r.
func UploadProjectFile(e Executor, projectID string, name string, fileName string, transform *FileTransformation, r io.Reader) error {

	b := new(bytes.Buffer)

	// IMPORTANT:
	// Since RexReference and ProjectFile is a 1..n relationship, we have
	// to create the RexReference before we create the ProjectFile

	// Query the project reference (required)
	rootReferenceURL := RexBaseURL + apiProjects + "/" + projectID + "/rootRexReference"
	req, _ := http.NewRequest("GET", rootReferenceURL, b)
	resp, err := e.Execute(req)
	if err != nil {
		return err
	}
	// Check if root reference is available, spit error if not!
	if resp.StatusCode != 200 {
		io.Copy(ioutil.Discard, resp.Body)
		return fmt.Errorf("Cannot create project file reference, because no project reference is set")
	}
	body, _ := ioutil.ReadAll(resp.Body)
	io.Copy(ioutil.Discard, resp.Body)

	parentReferenceURL := gjson.Get(string(body), "_links.self.href").String()

	// Create a RexReference as well
	uuid := uuid.New().String()
	rexReference := Reference{
		Project:         RexBaseURL + apiProjects + "/" + projectID,
		RootReference:   false,
		ParentReference: parentReferenceURL,
		Key:             uuid,
		FileTransform:   transform,
	}

	selfLink, err := createRexReference(e, &rexReference)
	if err != nil {
		return err
	}

	projectFile := struct {
		Name         string `json:"name"`
		Project      string `json:"project"`
		RexReference string `json:"rexReference"`
	}{
		Name:         name,
		Project:      RexBaseURL + apiProjects + "/" + projectID,
		RexReference: selfLink,
	}

	// Create project file
	json.NewEncoder(b).Encode(projectFile)
	req, _ = http.NewRequest("POST", RexBaseURL+apiProjectFiles, b)
	resp, err = e.Execute(req)
	if err != nil {
		return err
	}
	body, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 201 {
		io.Copy(ioutil.Discard, resp.Body)
		return fmt.Errorf("Got server status %d with error: %s ", resp.StatusCode, body)
	}

	// Upload the actual payload
	uploadURL := gjson.Get(string(body), "_links.file\\.upload.href").String()
	io.Copy(ioutil.Discard, resp.Body)
	return uploadFileContent(e, uploadURL, fileName, r)
}

func uploadFileContent(e Executor, uploadURL string, fileName string, r io.Reader) error {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fileName)
	io.Copy(part, r)
	writer.Close()

	req, _ := http.NewRequest("POST", uploadURL, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	resp, err := e.Execute(req)
	io.Copy(ioutil.Discard, resp.Body)
	return err
}

// UpdateProjectFile - test code
func updateProjectFile(c *Client) {
	var body = []byte(`{"type": "rex"}`)
	req, _ := http.NewRequest("PATCH", RexBaseURL+"/api/v2/projectFiles/1044", bytes.NewBuffer(body))
	c.Token.SetAuthHeader(req)
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Status)
}
