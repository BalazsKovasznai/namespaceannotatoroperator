package controllers

import (
	"context"
	"reflect"

	// Import necessary Kubernetes API packages
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	// Import custom API types
	devopsv1alpha1 "namespaceAnnotator/api/v1alpha1"
)

// NamespaceAnnotateReconciler reconciles a NamespaceAnnotate object
type NamespaceAnnotateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=devops.example.io,resources=namespaceannotates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devops.example.io,resources=namespaceannotates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devops.example.io,resources=namespaceannotates/finalizers,verbs=update

func (r *NamespaceAnnotateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Initialize NamespaceAnnotate and Namespace objects
	namesapceAnnotate := &devopsv1alpha1.NamespaceAnnotate{}
	namespace := &corev1.Namespace{}

	// Get the NamespaceAnnotate object with the specified name and namespace
	if err := r.Get(ctx, req.NamespacedName, namesapceAnnotate); err != nil {
		if errors.IsNotFound(err) {
			// If the NamespaceAnnotate object is not found, return success
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get namesapceAnnotate")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get the Namespace object associated with the NamespaceAnnotate object
	if err := r.Get(ctx, types.NamespacedName{Name: namesapceAnnotate.Namespace}, namespace); err != nil {
		logger.Error(err, "Failed to get namespace")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If the NamespaceAnnotate object is not marked for deletion
	if namesapceAnnotate.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(namesapceAnnotate, myFinalizerName) {
			// If the NamespaceAnnotate object does not contain the finalizer, add it
			if err := r.addMyFinalizer(ctx, *namesapceAnnotate, *namespace); err != nil {
				logger.Error(err, "Failed to add finalizer")
				return ctrl.Result{}, err
			}
			// Return success
			return ctrl.Result{}, nil
		}
	} else {
		// If the NamespaceAnnotate object is marked for deletion, handle it
		if err := r.handleDeletionMakred(ctx, *namesapceAnnotate, *namespace); err != nil {
			logger.Error(err, "Failed to cleanup namesapceAnnotate")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// get the keys to sync
	unConflictedKeys := fillterConflictedAnnotations(*namesapceAnnotate, *namespace)
	managedKeys := namesapceAnnotate.Status.SyncedAnnotations
	if err := r.syncNamespaceWithAnnotations(ctx, *namesapceAnnotate, unConflictedKeys, managedKeys, *namespace); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.setSyncedStatus(ctx, *namesapceAnnotate, unConflictedKeys); err != nil {
		return ctrl.Result{}, err
	}

	// return success
	return ctrl.Result{}, nil
}

// Set synced status in the object
func (r NamespaceAnnotateReconciler) setSyncedStatus(ctx context.Context, na devopsv1alpha1.NamespaceAnnotate, unConflictedKeys []string) error {
	logger := log.FromContext(ctx)
	if reflect.DeepEqual(na.Status.SyncedAnnotations, unConflictedKeys) {
		return nil
	}
	na.Status.SyncedAnnotations = unConflictedKeys
	if err := r.Status().Update(ctx, &na); err != nil {
		logger.Error(err, "Failed to set namesapceAnnotate status")
		return err
	}
	return nil
}

func (r NamespaceAnnotateReconciler) cleanManagedAnnotations(ctx context.Context, na devopsv1alpha1.NamespaceAnnotate, namespace corev1.Namespace) error {
	logger := log.FromContext(ctx)
	var annotationsToSet map[string]string = namespace.Annotations
	for _, k := range na.Status.SyncedAnnotations {
		delete(annotationsToSet, k)
	}
	namespace.SetAnnotations(annotationsToSet)
	if err := r.Update(ctx, &namespace); err != nil {
		logger.Error(err, "Failed to update Namespace")
	}
	return nil
}

// Object deletion flow
func (r NamespaceAnnotateReconciler) handleDeletionMakred(ctx context.Context, na devopsv1alpha1.NamespaceAnnotate, namespace corev1.Namespace) error {
	logger := log.FromContext(ctx)
	logger.Info("Start deleteing namesapceAnnotate")
	if controllerutil.ContainsFinalizer(&na, myFinalizerName) {
		if err := r.cleanManagedAnnotations(ctx, na, namespace); err != nil {
			return err
		}
		controllerutil.RemoveFinalizer(&na, myFinalizerName)
		if err := r.Update(ctx, &na); err != nil {
			logger.Error(err, "Failed to update namesapceAnnotate")
		}
	}
	logger.Info("namesapceAnnotate deleted successfully")
	return nil
}

// addMyFinalizer adds a finalizer to the NamespaceAnnotate object and updates it.
func (r NamespaceAnnotateReconciler) addMyFinalizer(ctx context.Context, na devopsv1alpha1.NamespaceAnnotate, namespace corev1.Namespace) error {
	logger := log.FromContext(ctx)
	controllerutil.AddFinalizer(&na, myFinalizerName)
	if err := r.Update(ctx, &na); err != nil {
		logger.Error(err, "Failed to update namesapceAnnotate")
		return err
	}
	return nil
}

// syncNamespaceWithAnnotations updates the Namespace object's annotations to match the NamespaceAnnotate object's annotations.
func (r NamespaceAnnotateReconciler) syncNamespaceWithAnnotations(ctx context.Context, na devopsv1alpha1.NamespaceAnnotate, unConflicted []string, managedKeys []string, namespace corev1.Namespace) error {
	logger := log.FromContext(ctx)
	var annotationsToSet map[string]string = make(map[string]string)
	// Copy annotations from Namespace object to annotationsToSet, excluding unConflicted and managedKeys annotations
	for k, v := range namespace.GetAnnotations() {
		if !slices.Contains(unConflicted, k) && slices.Contains(managedKeys, k) {
			continue
		}
		annotationsToSet[k] = v
	}
	for k, v := range na.Spec.Annotations {
		if slices.Contains(unConflicted, k) {
			annotationsToSet[k] = v
		}
	}
	// If Namespace object's annotations are already equal to annotationsToSet, return
	if reflect.DeepEqual(annotationsToSet, namespace.GetAnnotations()) {
		return nil
	}
	logger.Info("Syncing namesapceAnnotate with namespace")
	namespace.SetAnnotations(annotationsToSet)
	// Update the Namespace object
	if err := r.Update(ctx, &namespace); err != nil {
		logger.Error(err, "Failed to update Namespace")
		return err
	}
	logger.Info("Namespace is in sync with namesapceAnnotate")
	return nil
}

// Fetch the relevent Objects to reconcile after namespace change
func (r NamespaceAnnotateReconciler) findObjectsForNamespace(namespace client.Object) []reconcile.Request {
	attachedNamespacedAnnotations := &devopsv1alpha1.NamespaceAnnotateList{}
	listOps := &client.ListOptions{
		Namespace: namespace.GetName(),
	}
	err := r.List(context.TODO(), attachedNamespacedAnnotations, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedNamespacedAnnotations.Items))
	for i, item := range attachedNamespacedAnnotations.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceAnnotateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsv1alpha1.NamespaceAnnotate{}).
		Watches(
			&source.Kind{Type: &corev1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForNamespace),
		).
		Complete(r)
}
