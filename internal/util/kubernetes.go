package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/bootstrapper/internal/log"
)

// GetCluster creates and initializes a clusters.Cluster object based on the provided kubeconfigPath.
// If kubeconfigPath is empty, it tries to read the "KUBECONFIG" environment variable.
// If that is also empty, it defaults to "$HOME/.kube/config".
func GetCluster(kubeconfigPath, id string, scheme *runtime.Scheme) (*clusters.Cluster, error) {
	if len(kubeconfigPath) > 0 {
		return createCluster(kubeconfigPath, id, scheme)
	}

	kubeconfigEnvVar := os.Getenv("KUBECONFIG")
	if len(kubeconfigEnvVar) > 0 {
		return createCluster(kubeconfigEnvVar, id, scheme)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting user home directory: %w", err)
	}

	homeConfigPath := filepath.Join(homeDir, ".kube", "config")
	return createCluster(homeConfigPath, id, scheme)
}

func createCluster(kubeconfigPath, id string, scheme *runtime.Scheme) (*clusters.Cluster, error) {
	c := clusters.New(id)
	c.WithConfigPath(kubeconfigPath)

	err := c.InitializeRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("error initializing REST config: %w", err)
	}

	err = c.InitializeClient(scheme)
	if err != nil {
		return nil, fmt.Errorf("error initializing cluster client: %w", err)
	}

	return c, nil
}

func ApplyManifests(ctx context.Context, cluster *clusters.Cluster, manifests []byte) error {

	// Parse manifests into unstructured objects
	reader := bytes.NewReader(manifests)
	unstructuredObjects, err := parseManifests(reader)
	if err != nil {
		return fmt.Errorf("error parsing manifests: %w", err)
	}

	// Apply objects to the platform cluster
	for _, u := range unstructuredObjects {
		if err = applyUnstructuredObject(ctx, cluster, u); err != nil {
			return err
		}
	}

	return nil
}

func parseManifests(reader io.Reader) ([]*unstructured.Unstructured, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	var result []*unstructured.Unstructured
	for {
		u := &unstructured.Unstructured{}
		err := decoder.Decode(u)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		if len(u.Object) == 0 {
			continue
		}
		result = append(result, u)
	}
	return result, nil
}

func applyUnstructuredObject(ctx context.Context, cluster *clusters.Cluster, u *unstructured.Unstructured) error {
	logger := log.GetLogger()
	objectKey := client.ObjectKeyFromObject(u)
	objectLogString := fmt.Sprintf("%s %s", u.GetObjectKind().GroupVersionKind().String(), objectKey.String())

	existingObj := &unstructured.Unstructured{}
	existingObj.SetGroupVersionKind(u.GroupVersionKind())
	getErr := cluster.Client().Get(ctx, objectKey, existingObj)
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// create object
			logger.Tracef("Creating object %s", objectLogString)
			createErr := cluster.Client().Create(ctx, u)
			if createErr != nil {
				return fmt.Errorf("error creating object %s: %w", objectLogString, createErr)
			}
		} else {
			return fmt.Errorf("error reading object %s: %w", objectLogString, getErr)
		}
	} else {
		// update object
		logger.Tracef("Updating object %s", objectLogString)
		u.SetResourceVersion(existingObj.GetResourceVersion())
		updateErr := cluster.Client().Update(ctx, u)
		if updateErr != nil {
			return fmt.Errorf("error updating object %s: %w", objectLogString, updateErr)
		}
	}
	return nil
}
