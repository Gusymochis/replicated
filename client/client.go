// Package client manages channels and releases through the Replicated Vendor API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	apps "github.com/replicatedhq/replicated/gen/go/v1"
	channels "github.com/replicatedhq/replicated/gen/go/v1"
	releases "github.com/replicatedhq/replicated/gen/go/v1"
	v2 "github.com/replicatedhq/replicated/gen/go/v2"
)

const apiOrigin = "https://api.replicated.com/vendor"

// Client provides methods to manage apps, channels, and releases.
type Client interface {
	GetApp(slugOrID string) (*apps.App, error)
	CreateApp(opts *AppOptions) (*apps.App, error)

	ListChannels(appID string) ([]channels.AppChannel, error)
	CreateChannel(appID string, opts *ChannelOptions) ([]channels.AppChannel, error)
	ArchiveChannel(appID, channelID string) error
	GetChannel(appID, channelID string) (*channels.AppChannel, []channels.ChannelRelease, error)

	ListReleases(appID string) ([]releases.AppReleaseInfo, error)
	CreateRelease(appID string, opts *ReleaseOptions) (*releases.AppReleaseInfo, error)
	UpdateRelease(appID string, sequence int64, opts *ReleaseOptions) error
	GetRelease(appID string, sequence int64) (*releases.AppRelease, error)
	PromoteRelease(
		appID string,
		sequence int64,
		label string,
		notes string,
		required bool,
		channelIDs ...string) error

	CreateLicense(*v2.LicenseV2) (*v2.LicenseV2, error)
}

type AppOptions struct {
	Name string
}

type ChannelOptions struct {
	Name        string
	Description string
}

type ReleaseOptions struct {
	YAML string
}

// An HTTPClient communicates with the Replicated Vendor HTTP API.
type HTTPClient struct {
	apiKey    string
	apiOrigin string
}

// New returns a new  HTTP client.
func New(apiKey string) Client {
	c := &HTTPClient{
		apiKey:    apiKey,
		apiOrigin: apiOrigin,
	}

	return c
}

func NewHTTPClient(origin string, apiKey string) Client {
	c := &HTTPClient{
		apiKey:    apiKey,
		apiOrigin: origin,
	}

	return c
}

func (c *HTTPClient) doJSON(method, path string, successStatus int, reqBody, respBody interface{}) error {
	endpoint := fmt.Sprintf("%s%s", c.apiOrigin, path)
	var buf bytes.Buffer
	if reqBody != nil {
		if err := json.NewEncoder(&buf).Encode(reqBody); err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, endpoint, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != successStatus {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("%s %s %d: %s", method, endpoint, resp.StatusCode, body)
	}
	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("%s %s response decoding: %v", method, endpoint, err)
		}
	}
	return nil
}
