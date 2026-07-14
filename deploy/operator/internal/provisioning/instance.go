package provisioning

import (
	"fmt"
	"os/exec"
	"strings"
)

// InstanceProvisioner handles dedicated-mode IAM instance lifecycle.
// It provisions a complete GGID deployment via Helm in a dedicated namespace.
type InstanceProvisioner struct {
	helmChartPath string
}

// NewInstanceProvisioner creates a provisioner for dedicated IAM instances.
func NewInstanceProvisioner() *InstanceProvisioner {
	return &InstanceProvisioner{
		helmChartPath: "/charts/ggid",
	}
}

// ProvisionResult contains info about a provisioned instance.
type ProvisionResult struct {
	Namespace    string
	HelmRelease  string
	TenantID     string
}

// Provision deploys a dedicated GGID instance via Helm.
func (p *InstanceProvisioner) Provision(name, namespace string, replicas int32, dbDriver, dbHost string, dbPort int32, dbName, adminEmail string) (*ProvisionResult, error) {
	if namespace == "" {
		namespace = "ggid-" + name
	}

	// 1. Create namespace
	if err := runCommand("kubectl", "create", "namespace", namespace, "--dry-run=client", "-o", "yaml", "|", "kubectl", "apply", "-f", "-"); err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	// 2. Generate tenant UUID (using uuidgen or fallback)
	tenantID, err := runCommandCapture("uuidgen")
	if err != nil || tenantID == "" {
		tenantID = fmt.Sprintf("%s-tenant", name)
	}
	tenantID = strings.TrimSpace(tenantID)

	// 3. Helm install
	helmRelease := "ggid-" + name
	dbURL := fmt.Sprintf("%s://ggid:CHANGE_ME@%s:%d/%s?sslmode=require", dbDriver, dbHost, dbPort, dbName)

	values := []string{
		fmt.Sprintf("replicas=%d", replicas),
		fmt.Sprintf("database.driver=%s", dbDriver),
		fmt.Sprintf("database.url=%s", dbURL),
		fmt.Sprintf("tenant.id=%s", tenantID),
		fmt.Sprintf("admin.email=%s", adminEmail),
	}

	helmArgs := []string{"upgrade", "--install", helmRelease, p.helmChartPath,
		"--namespace", namespace,
		"--create-namespace",
		"--wait", "--timeout", "5m",
	}
	for _, v := range values {
		helmArgs = append(helmArgs, "--set", v)
	}

	if err := runCommand("helm", helmArgs...); err != nil {
		return nil, fmt.Errorf("helm install failed: %w", err)
	}

	return &ProvisionResult{
		Namespace:   namespace,
		HelmRelease: helmRelease,
		TenantID:    tenantID,
	}, nil
}

// Deprovision removes a dedicated GGID instance.
func (p *InstanceProvisioner) Deprovision(helmRelease, namespace string) error {
	// Uninstall Helm release
	if err := runCommand("helm", "uninstall", helmRelease, "--namespace", namespace); err != nil {
		return fmt.Errorf("helm uninstall: %w", err)
	}
	// Delete namespace
	if err := runCommand("kubectl", "delete", "namespace", namespace, "--ignore-not-found"); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}
	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), string(output))
	}
	return nil
}

func runCommandCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
