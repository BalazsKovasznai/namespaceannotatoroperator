# namesapceannotateor
The `Namespace Annotator` Kubernetes operator is designed to allow end-users to have greater control over user-defined annotations on their Kubernetes namespace. Annotations are key-value pairs that can be attached to Kubernetes resources, providing additional information that can be used by applications and tools.

## Description
The Namespace Annotator operator can be used to automatically add or modify annotations on a namespace. 
This can be useful for a variety of purposes, such as adding metadata to namespaces for better organization or controlling access to certain resources based on their annotations.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/namesapceannotateor:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/namesapceannotateor:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## usage 
first, create the resource `NamespaceAnnotate`.

```
apiVersion: devops.example.io/v1alpha1
kind: NamespaceAnnotate
metadata:
  name: example
  namespace: test-ann
spec: 
  annotations:
    "test": "test"
    "example": "examples"

```

and watch how its syncs with your namespace by applying your annotations.

if you want to edit the annotations simply edit the `NamespaceAnnotate` resource or delete it to get rid of them.
