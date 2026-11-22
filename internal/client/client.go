// Package client is a small client for the Staysa Cloud REST API (v1).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client calls the API of one Staysa Cloud console with an API token.
type Client struct {
	BaseURL   string // https://console.example.com
	Token     string // pla_...
	HTTP      *http.Client
	UserAgent string
}

// ErrNotFound is returned for a 404 answer: the object does not exist (any more).
var ErrNotFound = errors.New("not found")

// APIError is an error answer of the API.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("API error %d: %s", e.Status, e.Message) }

func New(baseURL, token, version string) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		Token:     token,
		HTTP:      &http.Client{Timeout: 60 * time.Second},
		UserAgent: "terraform-provider-staysa/" + version,
	}
}

// P escapes one path segment.
func P(s string) string { return url.PathEscape(s) }

// Do sends a request with an optional JSON body and decodes a JSON answer into out.
// Rate-limited answers (429) and gateway errors are retried a few times.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt*attempt) * time.Second):
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/api/v1"+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.UserAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusNotFound:
			return ErrNotFound
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusBadGateway ||
			resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout:
			lastErr = &APIError{resp.StatusCode, strings.TrimSpace(string(data))}
			continue
		case resp.StatusCode >= 300:
			var e struct {
				Error string `json:"error"`
			}
			msg := strings.TrimSpace(string(data))
			if json.Unmarshal(data, &e) == nil && e.Error != "" {
				msg = e.Error
			}
			return &APIError{resp.StatusCode, msg}
		}
		if out != nil && len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, out); err != nil {
				return fmt.Errorf("unexpected answer from %s %s: %w", method, path, err)
			}
		}
		return nil
	}
	return lastErr
}

// ---------- types (JSON field names are matched without regard to case) ----------

type Me struct {
	Organization string `json:"organization"`
	Role         string `json:"role"`
}

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Autoscaling struct {
	Min        int `json:"min"`
	Max        int `json:"max"`
	CPUPercent int `json:"cpu_percent"`
}

type Service struct {
	ID             string            `json:"id"`
	Project        string            `json:"project"`
	Image          string            `json:"image"`
	Port           int               `json:"port"`
	HealthPath     string            `json:"health_path"`
	CPUMillis      int               `json:"cpu_millis"`
	MemoryMB       int               `json:"memory_mb"`
	VolumePath     string            `json:"volume_path"`
	Instances      int               `json:"instances"`
	Worker         bool              `json:"worker"`
	Env            map[string]string `json:"env"`
	PublicPorts    []string          `json:"public_ports"`
	Autoscaling    *Autoscaling      `json:"autoscaling"`
	MetricsPath    string            `json:"metrics_path"`
	MetricsPort    int               `json:"metrics_port"`
	EdgeCache      bool              `json:"edge_cache"`
	Public         bool              `json:"public"`
	DefaultAddress bool              `json:"default_address"`
	InternalHost   string            `json:"internal_host"`
	URL            string            `json:"url"`
	Status         string            `json:"status"`
	Revision       int               `json:"revision"`
}

type Deployment struct {
	Revision int    `json:"revision"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type Database struct {
	ID            string `json:"id"`
	Project       string `json:"project"`
	Engine        string `json:"engine"`
	Version       string `json:"version"`
	Status        string `json:"status"`
	CPUMillis     int    `json:"cpu_millis"`
	MemoryMB      int    `json:"memory_mb"`
	BackupEnabled bool   `json:"backup_enabled"`
	BackupHour    int    `json:"backup_hour"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Database      string `json:"database"`
	Username      string `json:"username"`
	SecretName    string `json:"secret_name"`
	Password      string `json:"password"`
	URL           string `json:"url"`
}

type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Domain struct {
	Hostname   string      `json:"hostname"`
	Verified   bool        `json:"verified"`
	DNSRecords []DNSRecord `json:"dns_records"`
}

type Secret struct {
	Name string `json:"name"`
}

type ProjectVariable struct {
	Name   string `json:"name"`
	Secret bool   `json:"secret"`
	Value  string `json:"value"`
}

type Zone struct {
	Domain string `json:"domain"`
	Status string `json:"status"`
}

type ZoneList struct {
	Zones       []Zone   `json:"zones"`
	Nameservers []string `json:"nameservers"`
}

type ZoneRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Value     string `json:"value"`
	Priority  int    `json:"priority"`
	TTL       int    `json:"ttl"`
	ManagedBy string `json:"managedby"`
}

type ZoneRecords struct {
	Zone    string       `json:"zone"`
	Status  string       `json:"status"`
	Records []ZoneRecord `json:"records"`
}

type Bucket struct {
	Name     string `json:"name"`
	S3Name   string `json:"s3_name"`
	QuotaMB  int    `json:"quota_mb"`
	Website  string `json:"website_url"`
	Endpoint string `json:"endpoint"`
	Region   string `json:"region"`
}
