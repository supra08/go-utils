package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/keptn/go-utils/pkg/api/models"
)

// EventHandler handles services
type EventHandler struct {
	BaseURL    string
	AuthToken  string
	AuthHeader string
	HTTPClient *http.Client
	Scheme     string
}

// EventFilter allows to filter events based on the provided properties
type EventFilter struct {
	Project      	string
	Stage        	string
	Service      	string
	EventType    	string
	KeptnContext 	string
	EventID      	string
	PageSize		string
	NumberOfPages	int
}

// NewEventHandler returns a new EventHandler
func NewEventHandler(baseURL string) *EventHandler {
	if strings.Contains(baseURL, "https://") {
		baseURL = strings.TrimPrefix(baseURL, "https://")
	} else if strings.Contains(baseURL, "http://") {
		baseURL = strings.TrimPrefix(baseURL, "http://")
	}
	return &EventHandler{
		BaseURL:    baseURL,
		AuthHeader: "",
		AuthToken:  "",
		HTTPClient: &http.Client{Transport: getClientTransport()},
		Scheme:     "http",
	}
}

const mongodbDatastoreServiceBaseUrl = "mongodb-datastore"

// NewAuthenticatedEventHandler returns a new EventHandler that authenticates at the endpoint via the provided token
func NewAuthenticatedEventHandler(baseURL string, authToken string, authHeader string, httpClient *http.Client, scheme string) *EventHandler {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	httpClient.Transport = getClientTransport()

	baseURL = strings.TrimPrefix(baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(baseURL, mongodbDatastoreServiceBaseUrl) {
		baseURL += "/" + mongodbDatastoreServiceBaseUrl
	}

	return &EventHandler{
		BaseURL:    baseURL,
		AuthHeader: authHeader,
		AuthToken:  authToken,
		HTTPClient: httpClient,
		Scheme:     scheme,
	}
}

func (e *EventHandler) getBaseURL() string {
	return e.BaseURL
}

func (e *EventHandler) getAuthToken() string {
	return e.AuthToken
}

func (e *EventHandler) getAuthHeader() string {
	return e.AuthHeader
}

func (e *EventHandler) getHTTPClient() *http.Client {
	return e.HTTPClient
}

// GetEvents returns all events matching the properties in the passed filter object
func (e *EventHandler) GetEvents(filter *EventFilter) ([]*models.KeptnContextExtendedCE, *models.Error) {

	u, err := url.Parse(e.Scheme + "://" + e.getBaseURL() + "/event?")
	if err != nil {
		log.Fatal("error parsing url")
	}

	query := u.Query()

	if filter.Project != "" {
		query.Set("project", filter.Project)
	}
	if filter.Stage != "" {
		query.Set("stage", filter.Stage)
	}
	if filter.Service != "" {
		query.Set("service", filter.Service)
	}
	if filter.KeptnContext != "" {
		query.Set("keptnContext", filter.KeptnContext)
	}
	if filter.EventID != "" {
		query.Set("eventID", filter.EventID)
	}
	if filter.EventType != "" {
		query.Set("type", filter.EventType)
	}
	if filter.PageSize != "" {
		query.Set("pageSize", filter.PageSize)
	}

	u.RawQuery = query.Encode()

	return e.getEvents(u.String(), filter.NumberOfPages)
}

func (e *EventHandler) getEvents(uri string, numberOfPages int) ([]*models.KeptnContextExtendedCE, *models.Error) {
	events := []*models.KeptnContextExtendedCE{}
	nextPageKey := ""

	for {
		url, err := url.Parse(uri)
		if err != nil {
			return nil, buildErrorResponse(err.Error())
		}
		q := url.Query()
		if nextPageKey != "" {
			q.Set("nextPageKey", nextPageKey)
			url.RawQuery = q.Encode()
		}
		req, err := http.NewRequest("GET", url.String(), nil)
		req.Header.Set("Content-Type", "application/json")
		addAuthHeader(req, e)

		resp, err := e.HTTPClient.Do(req)
		if err != nil {
			return nil, buildErrorResponse(err.Error())
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, buildErrorResponse(err.Error())
		}

		if resp.StatusCode == 200 {
			received := &models.Events{}
			err = json.Unmarshal(body, received)
			if err != nil {
				return nil, buildErrorResponse(err.Error())
			}
			events = append(events, received.Events...)

			if received.NextPageKey == "" || received.NextPageKey == "0" {
				break
			}

			nextPageKeyInt, _ := strconv.Atoi(received.NextPageKey)

			if numberOfPages > 0 && nextPageKeyInt >= numberOfPages {
				break
			}

			nextPageKey = received.NextPageKey
		} else {
			var respErr models.Error
			err = json.Unmarshal(body, &respErr)
			if err != nil {
				return nil, buildErrorResponse(err.Error())
			}
			return nil, &respErr
		}
	}

	return events, nil
}