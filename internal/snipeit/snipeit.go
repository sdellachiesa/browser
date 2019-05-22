// Package snipeit provides a client for using the Snipe-IT API.
//
// The aim of the client is to implement only a subset of Snipe-IT API
// needed for the LTER project. Not more not less.
package snipeit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

// A Client manages communication with the Snipe-IT API.
type Client struct {
	client *http.Client // HTTP client used to communicate with the API.
	token  string       // Snipe-IT API personal API token.

	BaseURL *url.URL
}

// NewClient returns a new Snipe-IT API client with provided base URL.
// If base URL does not have a trailing slash, one is added automatically.
func NewClient(baseURL, token string) (*Client, error) {
	baseEndpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(baseEndpoint.Path, "/") {
		baseEndpoint.Path += "/"
	}

	c := new(Client)
	c.client = http.DefaultClient
	c.token = "Bearer " + token
	c.BaseURL = baseEndpoint
	return c, nil
}

func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.token)

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			decErr := json.NewDecoder(resp.Body).Decode(v)
			if decErr == io.EOF {
				decErr = nil // ignore EOF errors caused by empty response body
			}
			if decErr != nil {
				err = decErr
			}
		}
	}

	return resp, err
}

// addOptions adds the parameters in opt as URL query parameters to s. opt
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// LocationOptions specifies a subset of optional query
// parameters for listing locations.
type LocationOptions struct {
	// Search string
	Search string `url:"search,omitempty"`
}

// Location represents a Snipe-IT location.
type Location struct {
	ID             int        `json:"id,omitempty"`
	Name           string     `json:"name,omitempty"`
	Image          string     `json:"image,omitempty"`
	Address        string     `json:"address,omitempty"`
	Address2       string     `json:"address2,omitempty"`
	City           string     `json:"city,omitempty"`
	State          string     `json:"state,omitempty"`
	Country        string     `json:"country,omitempty"`
	Zip            string     `json:"zip,omitempty"`
	AssetsAssigned int        `json:"assigned_assets_count,omitempty"`
	Assets         int        `json:"assets_count,omitempty"`
	Users          int        `json:"users_count,omitempty"`
	Currency       string     `json:"currency,omitempty"`
	CreatedAt      Timestamp  `json:"created_at,omitempty"`
	UpdatedAt      Timestamp  `json:"updated_at,omitempty"`
	Parent         string     `json:"parent,omitempty"`
	Manager        string     `json:"manager,omitempty"`
	Children       []Location `json:"children,omitempty"`
	Actions        struct {
		Update bool
		Delete bool
	} `json:"available_actions,omitempty"`
}

// List locations
//
// Snipe-IT API doc: https://snipe-it.readme.io/reference#locations
func (c *Client) Locations(opt *LocationOptions) ([]*Location, *http.Response, error) {
	u, err := addOptions("locations", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := c.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Total int
		Rows  []*Location
	}
	resp, err := c.Do(req, &response)
	if err != nil {
		return nil, resp, err
	}

	return response.Rows, resp, nil
}

// HardwareOptions specifies a subset of optional query
// parameters for listing assets.
type HardwareOptions struct {
	LocationID int `url:"location_id,omitempty"`
}

// Hardware represents a Snipe-IT hardware object.
type Hardware struct {
	ID       int    `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	AssetTag string `json:"asset_tag,omitempty"`
	Serial   string `json:"serial,omitempty"`
	Model    struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"model,omitempty"`
	ModelNumber string `json:"model_number,omitempty"`
	StatusLabel struct {
		ID         int    `json:"id,omitempty"`
		Name       string `json:"name,omitempty"`
		StatusMeta string `json:"status_meta,omitempty"`
	} `json:"status_label,omitempty"`
	Category struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"category,omitempty"`
	Manufacturer struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"manufacturer,omitempty"`
	Supplier struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"supplier,omitempty"`
	Notes       string `json:"notes,omitempty"`
	OrderNumber string `json:"order_number,omitempty"`
	Company     string `json:"company,omitempty"`
	Location    int    `json:"location,omitempty"`
	RtdLocation struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"rtd_location,omitempty"`
	Image      string `json:"image,omitempty"`
	AssignedTo struct {
		ID        int    `json:"id,omitempty"`
		Username  string `json:"username,omitempty"`
		Name      string `json:"name,omitempty"`
		Firstname string `json:"first_name,omitempty"`
		Lastname  string `json:"last_name,omitempty"`
		Emplyee   string `json:"employee_number,omitempty"`
		Type      string `json:"type,omitempty"`
	} `json:"assigned_to,omitempty"`
	WarrantyMonths   interface{}   `json:"warranty_months,omitempty"`
	WarrantyExpires  interface{}   `json:"warranty_expires,omitempty"`
	CreatedAt        Timestamp     `json:"created_at,omitempty"`
	UpdatedAt        Timestamp     `json:"updated_at,omitempty"`
	DeletedAt        Timestamp     `json:"deleted_at,omitempty"`
	PurchaseDate     Timestamp     `json:"purchase_date,omitempty"`
	LastCheckout     Timestamp     `json:"last_checkout,omitempty"`
	ExpectedCheckin  Timestamp     `json:"expected_checkin,omitempty"`
	PurchaseCost     int           `json:"purchase_cost,omitempty"`
	UserCanCheckout  bool          `json:"user_can_checkout,omitempty"`
	CustomFields     []interface{} `json:"custom_fields,omitempty"`
	AvailableActions struct {
		Checkout bool `json:"checkout,omitempty"`
		Checkin  bool `json:"checkin,omitempty"`
		Clone    bool `json:"clone,omitempty"`
		Restore  bool `json:"restore,omitempty"`
		Update   bool `json:"update,omitempty"`
		Delete   bool `json:"delete,omitempty"`
	} `json:"available_actions,omitempty"`
}

// List Hardware
//
// https://snipe-it.readme.io/reference#hardware-list
func (c *Client) Hardware(opt *HardwareOptions) ([]*Hardware, *http.Response, error) {
	u, err := addOptions("hardware", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := c.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Total int
		Rows  []*Hardware
	}
	resp, err := c.Do(req, &response)
	if err != nil {
		return nil, resp, err
	}

	return response.Rows, resp, nil
}

// Timestamp is a custom time type for parsing Snipe-ITs
// API updated_at and created_at JSON values.
type Timestamp struct {
	time.Time
}

func (ts *Timestamp) UnmarshalJSON(b []byte) error {
	var d struct {
		Datetime string `json:"datetime"`
		Format   string `json:"formatted"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}

	const format = "2006-01-02 15:04:05"
	t, err := time.Parse(format, d.Datetime)
	if err != nil {
		return err
	}
	ts.Time = t
	return nil
}