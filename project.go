package rex

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

var (
	apiProject = "/api/v2/projects/"
)

// Project is the full structure of a REX project.
type Project struct {
	DateCreated string `json:"dateCreated"`
	CreatedBy   string `json:"createdBy"`
	LastUpdated string `json:"lastUpdated"`
	UpdatedBy   string `json:"updatedBy"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	TagLine     string `json:"tagLine"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Embedded    struct {
		RootRexReference struct {
			RootReference bool   `json:"rootReference"`
			Key           string `json:"key"`
			Links         struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"self"`
				Project struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"project"`
				ParentReference struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"parentReference"`
				ChildReferences struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"childReferences"`
				ProjectFiles struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"projectFiles"`
			} `json:"_links"`
		} `json:"rootRexReference"`
		ProjectFiles []struct {
			LastModified string `json:"lastModified"`
			FileSize     int    `json:"fileSize"`
			Name         string `json:"name"`
			Type         string `json:"type"`
			Links        struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"self"`
				RexReference struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"rexReference"`
				Project struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"project"`
				FileDownload struct {
					Href string `json:"href"`
				} `json:"file.download"`
			} `json:"_links"`
		} `json:"projectFiles"`
		RexReferences []struct {
			RootReference bool   `json:"rootReference"`
			Key           string `json:"key"`
			Links         struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"self"`
				Project struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"project"`
				ParentReference struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"parentReference"`
				ChildReferences struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"childReferences"`
				ProjectFiles struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"projectFiles"`
			} `json:"_links"`
		} `json:"rexReferences"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Project struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"project"`
		ThumbnailUpload struct {
			Href string `json:"href"`
		} `json:"thumbnail.upload"`
		ThumbnailDownload struct {
			Href string `json:"href"`
		} `json:"thumbnail.download"`
		ProjectFavorite struct {
			Href string `json:"href"`
		} `json:"projectFavorite"`
		RootRexReference struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"rootRexReference"`
		ProjectFiles struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"projectFiles"`
		ProjectAcls struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"projectAcls"`
		RexReferences struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"rexReferences"`
	} `json:"_links"`
}

// String nicely prints a project
func (p Project) String() string {

	hasRootRef := false
	if p.Embedded.RootRexReference.RootReference {
		hasRootRef = true
	}

	s := fmt.Sprintf("|------------------------------------------------------------------------------------------|\n")
	s += fmt.Sprintf("| Name           | %-71s |\n", p.Name)
	s += fmt.Sprintf("| Owner          | %-71s |\n", p.Owner)
	s += fmt.Sprintf("| Type           | %-71s |\n", p.Type)
	s += fmt.Sprintf("| Has root ref   | %-71t |\n", hasRootRef)
	s += fmt.Sprintf("| Total files    | %-71d |\n", len(p.Embedded.ProjectFiles))
	s += fmt.Sprintf("| Total refs     | %-71d |\n", len(p.Embedded.RexReferences))

	sz := 0
	for _, f := range p.Embedded.ProjectFiles {
		sz += f.FileSize
	}
	s += fmt.Sprintf("| Total size (KB)| %-71d |\n", sz/1024)
	s += fmt.Sprintf("|------------------------------------------------------------------------------------------|\n")

	for i, f := range p.Embedded.ProjectFiles {
		length := min(35, len(f.Name))
		s += fmt.Sprintf("| %3d | %-35s | %8d (kb) | %s |\n", i, f.Name[0:length], f.FileSize, f.LastModified)
	}
	s += fmt.Sprintf("|------------------------------------------------------------------------------------------|\n")

	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetProject retrieves the full project specified by the projectID (e.g. 1020)
func GetProject(e Executor, projectID string) (*Project, error) {
	req, _ := http.NewRequest("GET", RexBaseURL+apiProject+projectID, nil)

	resp, err := e.Execute(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
	}()

	var project Project
	err = json.NewDecoder(resp.Body).Decode(&project)
	return &project, err

}

// DownloadFile downloads a given link (e.g. project file link).
//
// The file name is anticipated by the provided information from the server
// using the content-disposition
func DownloadFile(e Executor, link string) error {
	req, _ := http.NewRequest("GET", link, nil)

	// Set content disposition in order to get information about the filename
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	response, err := e.Execute(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	contentInfo := response.Header.Get("Content-Disposition")

	fileName := "default.dat"
	re, _ := regexp.Compile("filename=\"(.*)\"")
	values := re.FindStringSubmatch(contentInfo)
	if len(values) > 0 {
		fileName = values[1]
	}

	output, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer output.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		return err
	}

	fmt.Println(n, "bytes downloaded and stored in", fileName, ".")
	return nil
}
