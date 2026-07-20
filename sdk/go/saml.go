package ggid

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"
)

// SAMLConfig holds the configuration for a SAML Service Provider.
// Applications use this to generate SP metadata for IdP integration.
type SAMLConfig struct {
	EntityID    string // SP Entity ID (e.g. "https://myapp.example.com/saml")
	ACSURL      string // Assertion Consumer Service URL (e.g. "https://myapp.example.com/saml/acs")
	SLOURL      string // Single Logout URL (optional)
	SignRequests bool  // Whether to sign SAML requests
}

// SPMetadata represents the SP metadata XML document.
type SPMetadata struct {
	XMLName      xml.Name `xml:"EntityDescriptor"`
	Xmlns        string   `xml:"xmlns,attr"`
	EntityID     string   `xml:"entityID,attr"`
	ValidUntil   string   `xml:"validUntil,attr,omitempty"`
	SPSSODescriptor struct {
		XMLName                  xml.Name `xml:"SPSSODescriptor"`
		ProtocolSupportEnumeration string `xml:"protocolSupportEnumeration,attr"`
		NameIDFormat             string   `xml:"NameIDFormat"`
		AssertionConsumerServices []AssertionConsumerService `xml:"AssertionConsumerService"`
		SingleLogoutServices     []SingleLogoutService `xml:"SingleLogoutService,omitempty"`
	} `xml:"SPSSODescriptor"`
}

// AssertionConsumerService represents the ACS binding element.
type AssertionConsumerService struct {
	XMLName  xml.Name `xml:"AssertionConsumerService"`
	Binding  string   `xml:"Binding,attr"`
	Location string   `xml:"Location,attr"`
	Index    int      `xml:"index,attr"`
}

// SingleLogoutService represents the SLO binding element.
type SingleLogoutService struct {
	XMLName  xml.Name `xml:"SingleLogoutService"`
	Binding  string   `xml:"Binding,attr"`
	Location string   `xml:"Location,attr"`
}

// GenerateSPMetadata generates SAML SP metadata XML for the given configuration.
// This metadata should be provided to the IdP administrator to configure the SAML connection.
//
// Usage:
//
//	cfg := &ggid.SAMLConfig{
//	    EntityID: "https://myapp.example.com/saml",
//	    ACSURL:   "https://myapp.example.com/saml/acs",
//	    SLOURL:   "https://myapp.example.com/saml/slo",
//	}
//	metadata, _ := ggid.GenerateSPMetadata(cfg)
//	// Write to file or serve at /saml/metadata endpoint
//	xml.NewEncoder(w).Encode(metadata)
func GenerateSPMetadata(cfg *SAMLConfig) (*SPMetadata, error) {
	if cfg == nil {
		return nil, fmt.Errorf("SAML config is nil")
	}
	if cfg.EntityID == "" {
		return nil, fmt.Errorf("entity ID is required")
	}
	if cfg.ACSURL == "" {
		return nil, fmt.Errorf("ACS URL is required")
	}

	meta := &SPMetadata{
		Xmlns:      "urn:oasis:names:tc:SAML:2.0:metadata",
		EntityID:   cfg.EntityID,
		ValidUntil: time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339),
	}

	meta.SPSSODescriptor.ProtocolSupportEnumeration = "urn:oasis:names:tc:SAML:2.0:protocol"
	meta.SPSSODescriptor.NameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	meta.SPSSODescriptor.AssertionConsumerServices = []AssertionConsumerService{
		{
			Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
			Location: cfg.ACSURL,
			Index:    0,
		},
	}

	if cfg.SLOURL != "" {
		meta.SPSSODescriptor.SingleLogoutServices = []SingleLogoutService{
			{
				Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
				Location: cfg.SLOURL,
			},
		}
	}

	return meta, nil
}

// FetchIdPMetadata retrieves IdP metadata XML from the GGID instance.
// Applications can use this to auto-configure the SAML connection.
//
// Usage:
//
//	client := ggid.NewClient("https://ggid.example.com")
//	idpMeta, _ := client.FetchIdPMetadata()
//	// Parse IdP certificate, SSO URL, Entity ID from metadata
func (c *Client) FetchIdPMetadata() ([]byte, error) {
	url := c.baseURL + "/saml/idp/metadata"
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IdP metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IdP metadata request failed: %s", resp.Status)
	}

	var buf []byte
	if _, err := resp.Body.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to read IdP metadata: %w", err)
	}

	return buf, nil
}

// ServeSPMetadata is an http.HandlerFunc that serves SP metadata XML.
// Applications can register it directly: `http.HandleFunc("/saml/metadata", ggid.ServeSPMetadata(cfg))`
//
// Usage:
//
//	cfg := &ggid.SAMLConfig{EntityID: "...", ACSURL: "..."}
//	http.HandleFunc("/saml/metadata", ggid.ServeSPMetadata(cfg))
func ServeSPMetadata(cfg *SAMLConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		meta, err := GenerateSPMetadata(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", "attachment; filename=\"sp-metadata.xml\"")
		xml.NewEncoder(w).Encode(meta)
	}
}