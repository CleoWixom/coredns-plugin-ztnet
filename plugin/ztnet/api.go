package ztnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Client communicates with the ZTNET REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient returns a Client with DefaultHTTPTimeout set.
func NewClient(baseURL, token string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, httpClient: &http.Client{Timeout: DefaultHTTPTimeout}}
}

// NetworkInfo holds v6 assignment mode flags.
type NetworkInfo struct {
	RFC4193  bool
	SixPlane bool
}

// Member is an authorised ZeroTier network member.
type Member struct {
	ID   string
	Name string
	IPs  []net.IP
}

type networkInfoResponse struct {
	V6AssignMode struct {
		SixPlane bool `json:"6plane"`
		RFC4193  bool `json:"rfc4193"`
	} `json:"v6AssignMode"`
}

type memberResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Authorized    bool     `json:"authorized"`
	IPAssignments []string `json:"ipAssignments"`
}

// GetNetworkInfo fetches v6AssignMode for networkID.
func (c *Client) GetNetworkInfo(ctx context.Context, networkID string) (*NetworkInfo, error) {
	url := fmt.Sprintf("%s/api/v1/network/%s/", c.baseURL, networkID)
	var response networkInfoResponse
	if err := c.getJSON(ctx, url, &response); err != nil {
		return nil, fmt.Errorf("ztnet: api: %w", err)
	}
	return &NetworkInfo{RFC4193: response.V6AssignMode.RFC4193, SixPlane: response.V6AssignMode.SixPlane}, nil
}

// GetMembers returns authorized==true members with IPv4-only IPs.
func (c *Client) GetMembers(ctx context.Context, networkID string) ([]Member, error) {
	url := fmt.Sprintf("%s/api/v1/network/%s/member/", c.baseURL, networkID)
	var response []memberResponse
	if err := c.getJSON(ctx, url, &response); err != nil {
		return nil, fmt.Errorf("ztnet: api: %w", err)
	}

	members := make([]Member, 0, len(response))
	for _, m := range response {
		if !m.Authorized {
			continue
		}
		member := Member{ID: strings.ToLower(m.ID), Name: strings.ReplaceAll(m.Name, " ", "_")}
		for _, assignment := range m.IPAssignments {
			ip := net.ParseIP(assignment)
			if ip == nil {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				member.IPs = append(member.IPs, ip4)
			}
		}
		members = append(members, member)
	}
	return members, nil
}

func (c *Client) getJSON(ctx context.Context, url string, dst interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return err
	}
	return nil
}
