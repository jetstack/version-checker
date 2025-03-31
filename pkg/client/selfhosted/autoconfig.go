package selfhosted

// import (
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"os"
// 	"strings"

// 	"github.com/go-logr/logr"
// 	"github.com/google/go-containerregistry/pkg/authn"
// 	"github.com/google/go-containerregistry/pkg/name"
// 	"github.com/google/go-containerregistry/pkg/v1/remote"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/rest"
// 	"k8s.io/client-go/tools/clientcmd"
// )

// // jwtClaims represents the relevant fields in the Kubernetes service account JWT
// type jwtClaims struct {
// 	Sub       string `json:"sub"`
// 	Namespace string `json:"kubernetes.io/serviceaccount/namespace"`
// 	Name      string `json:"kubernetes.io/serviceaccount/name"`
// }

// // getInClusterNamespaceAndServiceAccount extracts details from the service account JWT
// func getInClusterNamespaceAndServiceAccount(log logr.Logger) (string, string, error) {
// 	tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
// 	tokenBytes, err := os.ReadFile(tokenPath)
// 	if err != nil {
// 		log.Error(err, "Not running in-cluster or service account token missing")
// 		return "", "", err
// 	}

// 	parts := strings.Split(string(tokenBytes), ".")
// 	if len(parts) < 2 {
// 		return "", "", fmt.Errorf("invalid JWT token format")
// 	}

// 	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
// 	if err != nil {
// 		return "", "", fmt.Errorf("failed to decode JWT payload: %w", err)
// 	}

// 	var claims jwtClaims
// 	if err := json.Unmarshal(payload, &claims); err != nil {
// 		return "", "", fmt.Errorf("failed to parse JWT claims: %w", err)
// 	}

// 	if claims.Namespace == "" || claims.Name == "" {
// 		parts := strings.Split(claims.Sub, ":")
// 		if len(parts) == 4 && parts[0] == "system" && parts[1] == "serviceaccount" {
// 			claims.Namespace = parts[2]
// 			claims.Name = parts[3]
// 		}
// 	}

// 	if claims.Namespace == "" || claims.Name == "" {
// 		return "", "", fmt.Errorf("could not determine namespace/service account from JWT")
// 	}

// 	return claims.Namespace, claims.Name, nil
// }

// func getKubeConfigNamespace(log logr.Logger) (string, string, error) {
// 	kubeconfigPath := clientcmd.RecommendedHomeFile
// 	config, err := clientcmd.LoadFromFile(kubeconfigPath)
// 	if err != nil {
// 		log.Error(err, "Failed to load kubeconfig")
// 		return "", "", err
// 	}

// 	contextName := config.CurrentContext
// 	if contextName == "" {
// 		return "", "", fmt.Errorf("no active context in kubeconfig")
// 	}

// 	ctx, exists := config.Contexts[contextName]
// 	if !exists {
// 		return "", "", fmt.Errorf("context %s not found in kubeconfig", contextName)
// 	}

// 	if ctx.Namespace == "" {
// 		ctx.Namespace = "default"
// 	}

// 	return ctx.Namespace, ctx.AuthInfo, nil
// }

// func AutoConfig(log logr.Logger, serviceAccountOverride string) {
// 	log.Info("Starting container image authentication tool")

// 	inCluster := true
// 	namespace, serviceAccount, err := getInClusterNamespaceAndServiceAccount(log)
// 	if err != nil {
// 		log.Info("Not running in-cluster, falling back to kubeconfig")
// 		inCluster = false
// 	}

// 	if !inCluster {
// 		namespace, serviceAccount, err = getKubeConfigNamespace(log)
// 		if err != nil {
// 			log.Error(err, "Failed to determine namespace/service account from kubeconfig")
// 			os.Exit(1)
// 		}
// 	}

// 	if serviceAccountOverride != "" {
// 		serviceAccount = serviceAccountOverride
// 	}

// 	log.Info("Using configuration", "namespace", namespace, "serviceAccount", serviceAccount)

// 	// Get Kubernetes client
// 	var config *rest.Config
// 	if inCluster {
// 		config, err = rest.InClusterConfig()
// 	} else {
// 		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
// 	}
// 	if err != nil {
// 		log.Error(err, "Failed to create Kubernetes client")
// 		os.Exit(1)
// 	}

// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		log.Error(err, "Failed to create Kubernetes client")
// 		os.Exit(1)
// 	}

// 	log.Info("Successfully connected to Kubernetes cluster")

// 	// Define the image reference
// 	image := "gcr.io/my-project/my-image:latest"

// 	// Parse image reference
// 	ref, err := name.ParseReference(image)
// 	if err != nil {
// 		log.Error(err, "Failed to parse image reference")
// 		os.Exit(1)
// 	}

// 	// Fetch image metadata
// 	img, err := remote.Image(ref, remote.WithAuth(authn.DefaultKeychain))
// 	if err != nil {
// 		log.Error(err, "Failed to fetch image")
// 		os.Exit(1)
// 	}

// 	// Get image digest
// 	digest, err := img.Digest()
// 	if err != nil {
// 		log.Error(err, "Failed to get image digest")
// 		os.Exit(1)
// 	}

// 	log.Info("Successfully retrieved image digest", "digest", digest)
// }
