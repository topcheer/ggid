package mdm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// IntuneAdapter implements MDMConnector for Microsoft Intune via Microsoft Graph API.
type IntuneAdapter struct {
	Config ConnectorConfig
	client *http.Client
}

func (a *IntuneAdapter) ConnectorType() string { return "intune" }

func (a *IntuneAdapter) GetDevices(ctx context.Context) ([]Device, error) {
	url := a.Config.Endpoint + "/v1.0/deviceManagement/managedDevices"
	if a.Config.Endpoint == "" {
		url = "https://graph.microsoft.com/v1.0/deviceManagement/managedDevices"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	a.setAuth(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("intune API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("intune API returned %d", resp.StatusCode)
	}

	var body struct {
		Value []struct {
			ID                 string `json:"id"`
			DeviceName         string `json:"deviceName"`
			OSVersion          string `json:"osVersion"`
			OperatingSystem    string `json:"operatingSystem"`
			ComplianceState    string `json:"complianceState"` // "compliant"|"noncompliant"
			ManagedOwner       string `json:"managedOwner"`
			JailBroken         string `json:"jailBroken"` // "true"|"false"
			IsEncrypted        bool   `json:"isEncrypted"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("intune decode: %w", err)
	}

	var devices []Device
	for _, d := range body.Value {
		compliance := Unknown
		switch d.ComplianceState {
		case "compliant":
			compliance = Compliant
		case "noncompliant":
			compliance = NonCompliant
		}
		devices = append(devices, Device{
			DeviceID:         d.ID,
			OSVersion:        d.OSVersion,
			OS:               d.OperatingSystem,
			ComplianceStatus: compliance,
			Managed:          true,
			Jailbroken:       d.JailBroken == "true",
			Encrypted:        d.IsEncrypted,
		})
	}
	return devices, nil
}

func (a *IntuneAdapter) GetCompliance(ctx context.Context, deviceID string) (*Device, error) {
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.DeviceID == deviceID {
			return &d, nil
		}
	}
	return nil, nil
}

func (a *IntuneAdapter) GetPosture(ctx context.Context, deviceID string) (map[string]any, error) {
	d, err := a.GetCompliance(ctx, deviceID)
	if err != nil || d == nil {
		return nil, err
	}
	return map[string]any{
		"os_version": d.OSVersion,
		"encrypted":  d.Encrypted,
		"jailbroken": d.Jailbroken,
		"managed":    d.Managed,
		"source":     "intune",
	}, nil
}

func (a *IntuneAdapter) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.Config.AuthToken)
	req.Header.Set("Accept", "application/json")
}

// JamfAdapter implements MDMConnector for Jamf Pro via REST API.
type JamfAdapter struct {
	Config ConnectorConfig
	client *http.Client
}

func (a *JamfAdapter) ConnectorType() string { return "jamf" }

func (a *JamfAdapter) GetDevices(ctx context.Context) ([]Device, error) {
	url := a.Config.Endpoint + "/api/v1/computers-inventory"
	if a.Config.Endpoint == "" {
		url = "https://your-instance.jamfcloud.com/api/v1/computers-inventory"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	a.setAuth(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jamf API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jamf API returned %d", resp.StatusCode)
	}

	var body struct {
		Results []struct {
			ID            int    `json:"id"`
			UDID          string `json:"udid"`
			OSVersion     string `json:"osVersion"`
			Platform      string `json:"platform"`
			Managed       bool   `json:"managed"`
			EnrolledVia   string `json:"enrolledVia"`
			FileVault2     bool   `json:"fileVault2Enabled"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("jamf decode: %w", err)
	}

	var devices []Device
	for _, d := range body.Results {
		devices = append(devices, Device{
			DeviceID:         fmt.Sprintf("jamf-%d", d.ID),
			OSVersion:        d.OSVersion,
			OS:               d.Platform,
			ComplianceStatus: Compliant, // Jamf managed = compliant by default
			Managed:          d.Managed,
			Encrypted:        d.FileVault2,
		})
	}
	return devices, nil
}

func (a *JamfAdapter) GetCompliance(ctx context.Context, deviceID string) (*Device, error) {
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.DeviceID == deviceID {
			return &d, nil
		}
	}
	return nil, nil
}

func (a *JamfAdapter) GetPosture(ctx context.Context, deviceID string) (map[string]any, error) {
	d, err := a.GetCompliance(ctx, deviceID)
	if err != nil || d == nil {
		return nil, err
	}
	return map[string]any{
		"os_version": d.OSVersion,
		"encrypted":  d.Encrypted,
		"managed":    d.Managed,
		"source":     "jamf",
	}, nil
}

func (a *JamfAdapter) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.Config.AuthToken)
	req.Header.Set("Accept", "application/json")
}

// AndroidAdapter implements MDMConnector for Google Android Management API.
type AndroidAdapter struct {
	Config ConnectorConfig
	client *http.Client
}

func (a *AndroidAdapter) ConnectorType() string { return "android_management" }

func (a *AndroidAdapter) GetDevices(ctx context.Context) ([]Device, error) {
	url := a.Config.Endpoint + "/v1/enterprises/" + a.Config.TenantID + "/devices"
	if a.Config.Endpoint == "" {
		url = "https://androidmanagement.googleapis.com/v1/enterprises/" + a.Config.TenantID + "/devices"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	a.setAuth(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("android API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("android API returned %d", resp.StatusCode)
	}

	var body struct {
		Devices []struct {
			Name            string `json:"name"` // enterprises/{id}/devices/{id}
			DeviceSettings  struct {
				EncryptionStatus string `json:"encryptionStatus"`
			} `json:"deviceSettings"`
			State           string `json:"state"` // "ACTIVE"|"DISABLED"
			OSVersion       string `json:"osVersion"`
		} `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("android decode: %w", err)
	}

	var devices []Device
	for _, d := range body.Devices {
		compliance := NonCompliant
		if d.State == "ACTIVE" && d.DeviceSettings.EncryptionStatus == "ENCRYPTED" {
			compliance = Compliant
		}
		devices = append(devices, Device{
			DeviceID:         d.Name,
			OS:               "Android",
			OSVersion:        d.OSVersion,
			ComplianceStatus: compliance,
			Managed:          true,
			Encrypted:        d.DeviceSettings.EncryptionStatus == "ENCRYPTED",
		})
	}
	return devices, nil
}

func (a *AndroidAdapter) GetCompliance(ctx context.Context, deviceID string) (*Device, error) {
	devices, err := a.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.DeviceID == deviceID {
			return &d, nil
		}
	}
	return nil, nil
}

func (a *AndroidAdapter) GetPosture(ctx context.Context, deviceID string) (map[string]any, error) {
	d, err := a.GetCompliance(ctx, deviceID)
	if err != nil || d == nil {
		return nil, err
	}
	return map[string]any{
		"os_version": d.OSVersion,
		"encrypted":  d.Encrypted,
		"managed":    d.Managed,
		"source":     "android_management",
	}, nil
}

func (a *AndroidAdapter) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.Config.AuthToken)
}
