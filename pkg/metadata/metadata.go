package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var metadataBaseURL = "https://api.metadata.cloud.ibm.com"

var metadataAPIVersion = "2024-11-12"

type MetadataTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type IAMTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type VirtualNetworkInterfaceResponse struct {
	VirtualNetworkInterfaces []struct {
		ID  string `json:"id"`
		IPs []struct {
			Address string `json:"address"`
			ID      string `json:"id"`
		} `json:"ips"`
		Subnet struct {
			CRN string `json:"crn"`
			ID  string `json:"id"`
		}
		PrimaryIP struct {
			Address string `json:"address"`
		} `json:"primary_ip"`
	} `json:"virtual_network_interfaces"`
}

type InstanceResponse struct {
	Zone struct {
		Name string `json:"name"`
	} `json:"zone"`
}

type MetadataInterface interface {
	GetMetadataAccessToken() (string, error)
	GetIAMAccessToken(auth string) (string, error)
	GetMetadataVNIs(token string) (*VirtualNetworkInterfaceResponse, error)
	GetInstance(auth string) (*InstanceResponse, error)
}

type Metadata struct {
	client *http.Client
}

type MetadataRequest struct {
	Method string
	URL    string
	Auth   string
	Body   []byte
	Out    interface{}
}

func NewMetadataService() MetadataInterface {
	return &Metadata{client: &http.Client{}}
}

func (md *Metadata) GetMetadataAccessToken() (string, error) {
	var tokenResp MetadataTokenResponse
	err := md.doRequest(&MetadataRequest{
		Method: "PUT",
		URL:    fmt.Sprintf("%s/instance_identity/v1/token?version=%s", metadataBaseURL, metadataAPIVersion),
		Body:   []byte(`{"expires_in": 3600}`),
		Out:    &tokenResp,
	})
	if err != nil {
		return "", err
	}
	return tokenResp.AccessToken, nil
}

func (md *Metadata) GetIAMAccessToken(auth string) (string, error) {
	var tokenResp IAMTokenResponse
	err := md.doRequest(&MetadataRequest{
		Method: "POST",
		URL:    fmt.Sprintf("%s/instance_identity/v1/iam_token?version=%s", metadataBaseURL, metadataAPIVersion),
		Auth:   auth,
		Out:    &tokenResp,
	})
	if err != nil {
		return "", err
	}
	return tokenResp.AccessToken, nil
}

func (md *Metadata) GetMetadataVNIs(auth string) (*VirtualNetworkInterfaceResponse, error) {
	var metaResp VirtualNetworkInterfaceResponse
	err := md.doRequest(&MetadataRequest{
		Method: "GET",
		URL:    fmt.Sprintf("%s/metadata/v1/virtual_network_interfaces?version=%s", metadataBaseURL, metadataAPIVersion),
		Auth:   auth,
		Out:    &metaResp,
	})
	return &metaResp, err
}

func (md *Metadata) GetInstance(auth string) (*InstanceResponse, error) {
	var instResponse InstanceResponse
	err := md.doRequest(&MetadataRequest{
		Method: "GET",
		URL:    fmt.Sprintf("%s/metadata/v1/instance?version=%s", metadataBaseURL, metadataAPIVersion),
		Auth:   auth,
		Out:    &instResponse,
	})
	return &instResponse, err
}

func (md *Metadata) doRequest(reqData *MetadataRequest) error {
	req, err := http.NewRequest(reqData.Method, reqData.URL, bytes.NewBuffer(reqData.Body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Metadata-Flavor", "ibm")
	if reqData.Auth != "" {
		req.Header.Set("Authorization", "Bearer "+reqData.Auth)
	}

	resp, err := md.client.Do(req)
	if err != nil {
		return fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body error: %w", err)
	}

	if err := json.Unmarshal(respBody, reqData.Out); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}
	return nil
}
