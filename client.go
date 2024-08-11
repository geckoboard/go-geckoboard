package geckoboard

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

var (
	errUnexpectedResponse = errors.New("Sorry, there seems to be a problem with " +
		"Geckoboard's servers. Please try again, or check" +
		"https://geckoboard.statuspage.io")
)

type Client struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func New(apikey string) *Client {
	return NewWithURL(apikey, "https://api.geckoboard.com")
}

func NewWithURL(apikey, baseURL string) *Client {
	return &Client{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
		apiKey:  apikey,
	}
}

func (c *Client) DatasetService() DatasetService {
	return &datasetService{
		client:           c,
		maxRecordsPerReq: 500,
		jsonMarshalFn:    json.Marshal,
	}
}

func (c *Client) buildRequest(method, path string, body io.Reader) (*http.Request, error) {
	r, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	r.SetBasicAuth(c.apiKey, "")

	if body != nil {
		r.Header.Add("Content-Type", "application/json")
	}

	return r, nil
}

func (c *Client) doRequest(req *http.Request) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		return err
	}

	return nil
}

func (c *Client) checkResponse(resp *http.Response) error {
	if resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		return errUnexpectedResponse
	}

	gerr := &Error{StatusCode: resp.StatusCode}

	if err := json.NewDecoder(resp.Body).Decode(&gerr); err != nil {
		return err
	}

	return gerr
}
