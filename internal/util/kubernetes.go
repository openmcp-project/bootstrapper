package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	sigsyaml "sigs.k8s.io/yaml"

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
	unstructuredObjects, err := ParseManifests(reader)
	if err != nil {
		return fmt.Errorf("error parsing manifests: %w", err)
	}

	// Apply objects to the platform cluster
	for _, u := range unstructuredObjects {
		if err = CreateOrUpdate(ctx, cluster, u); err != nil {
			return err
		}
	}

	return nil
}

func ParseManifests(reader io.Reader) ([]*unstructured.Unstructured, error) {
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

func CreateOrUpdate(ctx context.Context, cluster *clusters.Cluster, obj client.Object) error {
	logger := log.GetLogger()
	objectKey := client.ObjectKeyFromObject(obj)
	objectLogString := fmt.Sprintf("%s %s", obj.GetObjectKind().GroupVersionKind().String(), objectKey.String())

	existing := obj.DeepCopyObject().(client.Object)
	err := cluster.Client().Get(ctx, client.ObjectKeyFromObject(obj), existing)

	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Tracef("Creating object %s", objectLogString)
			return cluster.Client().Create(ctx, obj)
		}
		return err
	}

	logger.Tracef("Updating object %s", objectLogString)
	obj.SetResourceVersion(existing.GetResourceVersion())
	return cluster.Client().Update(ctx, obj)
}

func PrintUnstructuredObjects(objects []*unstructured.Unstructured, writer io.Writer) error {
	for i, obj := range objects {
		// Add separator between objects (except before the first one)
		if i > 0 {
			if _, err := writer.Write([]byte("---\n")); err != nil {
				return fmt.Errorf("error writing separator: %w", err)
			}
		}

		// Convert unstructured object to YAML
		yamlBytes, err := sigsyaml.Marshal(obj.Object)
		if err != nil {
			return fmt.Errorf("error marshaling object to YAML: %w", err)
		}

		// Write YAML to writer
		if _, err := writer.Write(yamlBytes); err != nil {
			return fmt.Errorf("error writing YAML: %w", err)
		}
	}

	return nil
}
