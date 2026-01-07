package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL        string
	InstanceKey    string
	InstanceSecret string
	httpClient     *http.Client
}

type registerInstanceReq struct {
	Name string `json:"name"`
}

type registerInstanceResp struct {
	InstanceKey    string `json:"instance_key"`
	InstanceSecret string `json:"instance_secret"`
}

func RegisterInstance(baseURL, name string) (string, string, error) {
	body := registerInstanceReq{Name: name}
	b, err := json.Marshal(body)
	if err != nil {
		return "", "", err
	}

	resp, err := http.Post(baseURL+"/api/instances/register", "application/json", bytes.NewReader(b))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("http %d: %s", resp.StatusCode, string(data))
	}

	var out registerInstanceResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.InstanceKey, out.InstanceSecret, nil
}

func NewClient(baseURL, key, secret string) *Client {
	return &Client{
		BaseURL:        baseURL,
		InstanceKey:    key,
		InstanceSecret: secret,
		httpClient:     &http.Client{},
	}
}

func (c *Client) do(method, path string, body any, out any) error {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.InstanceKey != "" {
		req.Header.Set("X-Instance-Key", c.InstanceKey)
		req.Header.Set("X-Instance-Secret", c.InstanceSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) ListMonitors() ([]Monitor, error) {
	var ms []Monitor
	err := c.do("GET", "/api/monitors", nil, &ms)
	return ms, err
}

type CreateMonitorReq struct {
	Name             string  `json:"name"`
	URL              string  `json:"url"`
	CSSSelector      *string `json:"css_selector,omitempty"`
	FrequencySeconds int     `json:"frequency_seconds"`
	NotifyEmail      bool    `json:"notify_email"`
	NotifyEmailAddr  string  `json:"notify_email_address"`
}

type UpdateMonitorReq struct {
	FrequencySeconds int  `json:"frequency_seconds"`
	Active           bool `json:"active"`
}

func (c *Client) CreateMonitor(req CreateMonitorReq) (*Monitor, error) {
	var m Monitor
	err := c.do("POST", "/api/monitors", req, &m)
	return &m, err
}

func (c *Client) UpdateMonitor(id uint64, req UpdateMonitorReq) (*Monitor, error) {
	var m Monitor
	path := fmt.Sprintf("/api/monitors/%d", id)
	err := c.do("PUT", path, req, &m)
	return &m, err
}

func (c *Client) DeleteMonitor(id uint64) error {
	return c.do("DELETE", fmt.Sprintf("/api/monitors/%d", id), nil, nil)
}

func (c *Client) ListChanges(monitorID uint64) ([]ChangeEvent, error) {
	var out []ChangeEvent
	path := fmt.Sprintf("/api/monitors/%d/changes?limit=50", monitorID)
	err := c.do("GET", path, nil, &out)
	return out, err
}
