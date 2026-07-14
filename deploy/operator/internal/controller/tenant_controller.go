package controller

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iamv1alpha1 "github.com/ggid/ggid/deploy/operator/api/v1alpha1"
	"github.com/ggid/ggid/deploy/operator/internal/provisioning"
)

type GGIDTenantReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Provisioner *provisioning.TenantProvisioner
}

func (r *GGIDTenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tenant iamv1alpha1.GGIDTenant
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !tenant.DeletionTimestamp.IsZero() {
		logger.Info("deleting tenant", "tenantId", tenant.Status.TenantID)
		if tenant.Status.TenantID != "" {
			_ = r.Provisioner.Deprovision(tenant.Status.TenantID)
		}
		return ctrl.Result{}, nil
	}

	// Skip if already ready
	if tenant.Status.Phase == "Ready" {
		return ctrl.Result{}, nil
	}

	// Provision
	logger.Info("provisioning shared tenant", "name", tenant.Spec.TenantName, "slug", tenant.Spec.Slug)
	tenant.Status.Phase = "Provisioning"
	_ = r.Status().Update(ctx, &tenant)

	result, err := r.Provisioner.Provision(&provisioning.CreateTenantRequest{
		Name:     tenant.Spec.TenantName,
		Slug:     tenant.Spec.Slug,
		Plan:     tenant.Spec.Plan,
		Status:   "active",
		MaxUsers: tenant.Spec.MaxUsers,
	})
	if err != nil {
		tenant.Status.Phase = "Failed"
		logger.Error(err, "failed to provision tenant")
		_ = r.Status().Update(ctx, &tenant)
		return ctrl.Result{RequeueAfter: 30}, nil
	}

	// Seed default roles
	_ = r.Provisioner.SeedDefaultRoles(result.ID)

	// Create admin user
	if tenant.Spec.AdminEmail != "" {
		_ = r.Provisioner.CreateAdminUser(result.ID, tenant.Spec.AdminEmail, tenant.Spec.AdminPassword)
	}

	tenant.Status.Phase = "Ready"
	tenant.Status.TenantID = result.ID
	tenant.Status.GatewayURL = r.Provisioner.GatewayURL()
	if err := r.Status().Update(ctx, &tenant); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("tenant provisioned", "tenantId", result.ID)
	return ctrl.Result{}, nil
}

func (r *GGIDTenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1alpha1.GGIDTenant{}).
		Complete(r)
}
