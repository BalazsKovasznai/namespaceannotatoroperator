package controllers

import (
	devopsv1alpha1 "namespaceAnnotator/api/v1alpha1" // Importing the custom namespaceAnnotator API package

	"golang.org/x/exp/slices"   // Importing the slices package from the Golang.org external repository
	corev1 "k8s.io/api/core/v1" // Importing the Kubernetes core v1 API package
)

const myFinalizerName = "example.domain/finalizer" // Declaring a constant variable for a finalizer name

// isMapContaines is a utility function to check if map containes key
func isMapContaines(dict map[string]string, key string) bool {
	if _, ok := dict[key]; ok {
		return true
	}
	return false
}

// fillterConflictedAnnotations is a function to filter conflicted and un-conflicted annotations from a NamespaceAnnotate object and a Kubernetes namespace object
func fillterConflictedAnnotations(na devopsv1alpha1.NamespaceAnnotate, namespace corev1.Namespace) []string {
	var conflictedKeys []string
	var unConflictedKeys []string
	for k := range na.Spec.Annotations {
		if isMapContaines(namespace.GetAnnotations(), k) && !slices.Contains(na.Status.SyncedAnnotations, k) {
			conflictedKeys = append(conflictedKeys, k)
		} else {
			unConflictedKeys = append(unConflictedKeys, k)
		}
	}
	return unConflictedKeys
}
