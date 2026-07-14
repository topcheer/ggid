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

type GGIDInstanceReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Provisioner *provisioning.InstanceProvisioner
}

func (r *GGIDInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var instance iamv1alpha1.GGIDInstance
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !instance.DeletionTimestamp.IsZero() {
		logger.Info("deleting dedicated instance", "release", instance.Status.HelmRelease)
		if instance.Status.HelmRelease != "" {
			_ = r.Provisioner.Deprovision(instance.Status.HelmRelease, instance.Status.Namespace)
		}
		return ctrl.Result{}, nil
	}

	// Skip if already ready
	if instance.Status.Phase == "Ready" {
		return ctrl.Result{}, nil
	}

	// Provision
	logger.Info("provisioning dedicated instance", "name", instance.Name, "org", instance.Spec.OrganizationName)
	instance.Status.Phase = "Provisioning"
	_ = r.Status().Update(ctx, &instance)

	result, err := r.Provisioner.Provision(
		instance.Name,
		instance.Spec.Namespace,
		instance.Spec.Replicas,
		instance.Spec.Database.Driver,
		instance.Spec.Database.Host,
		instance.Spec.Database.Port,
		instance.Spec.Database.Name,
		instance.Spec.AdminEmail,
	)
	if err != nil {
		instance.Status.Phase = "Failed"
		logger.Error(err, "failed to provision instance")
		_ = r.Status().Update(ctx, &instance)
		return ctrl.Result{RequeueAfter: 60}, nil
	}

	instance.Status.Phase = "Ready"
	instance.Status.Namespace = result.Namespace
	instance.Status.HelmRelease = result.HelmRelease
	instance.Status.TenantID = result.TenantID
	if err := r.Status().Update(ctx, &instance); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("dedicated instance provisioned", "namespace", result.Namespace, "tenantId", result.TenantID)
	return ctrl.Result{}, nil
}

func (r *GGIDInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iamv1alpha1.GGIDInstance{}).
		Complete(r)
}
