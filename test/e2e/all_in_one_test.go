//+build e2e_tests

package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver/v4"
	"github.com/sethvargo/go-password/password"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kong/kubernetes-testing-framework/pkg/clusters"
	"github.com/kong/kubernetes-testing-framework/pkg/clusters/addons/metallb"
	"github.com/kong/kubernetes-testing-framework/pkg/environments"
	"github.com/kong/kubernetes-testing-framework/pkg/utils/kubernetes/generators"

	"github.com/kong/kubernetes-ingress-controller/internal/annotations"
)

// -----------------------------------------------------------------------------
// All-In-One Manifest Tests - Vars
// -----------------------------------------------------------------------------

const (
	// kongComponentWait is the maximum amount of time to wait for components (such as
	// the ingress controller or the Kong Gateway) to become responsive after
	// deployment to the cluster has finished.
	kongComponentWait = time.Minute * 5

	// ingressWait is the maximum amount of time to wait for a basic HTTP service
	// (e.g. httpbin) to come online and for ingress to have properly configured
	// proxy traffic to route to it.
	ingressWait = time.Minute * 3
)

var (
	// clusterVersionStr indicates the Kubernetes cluster version to use when
	// generating a testing environment and allows the caller to provide a specific
	// version. If no version is provided the default version for the cluster
	// provisioner in the testing framework will be used.
	clusterVersionStr = os.Getenv("KONG_CLUSTER_VERSION")

	// enterpriseLicenseSecretYAMLVar is the name of the ENV var used to pass an
	// enterprise license to the tests.
	enterpriseLicenseSecretYAMLVar = "KONG_ENTERPRISE_LICENSE_SECRET"

	// enterpriseLicenseSecretYAML is the full YAML manifest of a license secret needed
	// in order to support an enterprise deployment of Kong via the KIC.
	// This value must be provided via the environment for enterprise tests.
	enterpriseLicenseSecretYAML = os.Getenv(enterpriseLicenseSecretYAMLVar)
)

// -----------------------------------------------------------------------------
// All-In-One Manifest Tests - Suite
//
// The following tests ensure that the local "all-in-one" style deployment manifests
// (which are predominantly used for testing, whereas the helm chart is meant for
// production use cases) are functional by deploying them to a cluster and verifying
// some of the fundamental functionality of the ingress controller and the proxy to
// ensure that things are up and running.
// -----------------------------------------------------------------------------

const dblessPath = "../../deploy/single-v2/all-in-one-dbless.yaml"

func TestDeployAllInOneDBLESS(t *testing.T) {
	t.Log("configuring all-in-one-dbless.yaml manifest test")
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("building test cluster and environment")
	builder := environments.NewBuilder().WithAddons(metallb.New())
	if clusterVersionStr != "" {
		clusterVersion, err := semver.Parse(clusterVersionStr)
		require.NoError(t, err)
		builder.WithKubernetesVersion(clusterVersion)
	}
	env, err := builder.Build(ctx)
	require.NoError(t, err)
	defer env.Cleanup(ctx)

	t.Log("deploying kong components")
	deployKong(ctx, t, env, dblessPath)

	t.Log("running ingress tests to verify all-in-one deployed ingress controller and proxy are functional")
	verifyIngress(ctx, t, env)
}

const entDBLESSPath = "../../deploy/single-v2/all-in-one-enterprise-dbless.yaml"

func TestDeployAllInOneEnterpriseDBLESS(t *testing.T) {
	t.Log("configuring all-in-one-enterprise-dbless.yaml manifest test")
	if enterpriseLicenseSecretYAML == "" {
		t.Skipf("no license available to test enterprise: %s was not provided", enterpriseLicenseSecretYAMLVar)
	}
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("building test cluster and environment")
	builder := environments.NewBuilder().WithAddons(metallb.New())
	if clusterVersionStr != "" {
		clusterVersion, err := semver.Parse(clusterVersionStr)
		require.NoError(t, err)
		builder.WithKubernetesVersion(clusterVersion)
	}
	env, err := builder.Build(ctx)
	require.NoError(t, err)
	defer env.Cleanup(ctx)

	t.Log("generating a superuser password")
	adminPassword, adminPasswordSecretYAML, err := generateAdminPasswordSecret()
	require.NoError(t, err)

	t.Log("deploying kong components")
	deployKong(ctx, t, env, entDBLESSPath, enterpriseLicenseSecretYAML, adminPasswordSecretYAML)

	t.Log("running ingress tests to verify all-in-one deployed ingress controller and proxy are functional")
	verifyIngress(ctx, t, env)

	t.Log("verifying enterprise mode was enabled properly")
	verifyEnterprise(ctx, t, env, adminPassword)
}

const postgresPath = "../../deploy/single-v2/all-in-one-postgres.yaml"

func TestDeployAllInOnePostgres(t *testing.T) {
	t.Log("configuring all-in-one-postgres.yaml manifest test")
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("building test cluster and environment")
	builder := environments.NewBuilder().WithAddons(metallb.New())
	if clusterVersionStr != "" {
		clusterVersion, err := semver.Parse(clusterVersionStr)
		require.NoError(t, err)
		builder.WithKubernetesVersion(clusterVersion)
	}
	env, err := builder.Build(ctx)
	require.NoError(t, err)
	defer env.Cleanup(ctx)

	t.Log("deploying kong components")
	deployKong(ctx, t, env, postgresPath)

	t.Log("this deployment used a postgres backend, verifying that postgres migrations ran properly")
	verifyPostgres(ctx, t, env)

	t.Log("running ingress tests to verify all-in-one deployed ingress controller and proxy are functional")
	verifyIngress(ctx, t, env)
}

const entPostgresPath = "../../deploy/single-v2/all-in-one-enterprise-postgres.yaml"

func TestDeployAllInOneEnterprisePostgres(t *testing.T) {
	t.Log("configuring all-in-one-enterprise-postgres.yaml manifest test")
	if enterpriseLicenseSecretYAML == "" {
		t.Skipf("no license available to test enterprise: %s was not provided", enterpriseLicenseSecretYAMLVar)
	}
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("building test cluster and environment")
	builder := environments.NewBuilder().WithAddons(metallb.New())
	if clusterVersionStr != "" {
		clusterVersion, err := semver.Parse(clusterVersionStr)
		require.NoError(t, err)
		builder.WithKubernetesVersion(clusterVersion)
	}
	env, err := builder.Build(ctx)
	require.NoError(t, err)
	defer env.Cleanup(ctx)

	t.Log("generating a superuser password")
	adminPassword, adminPasswordSecretYAML, err := generateAdminPasswordSecret()
	require.NoError(t, err)

	t.Log("deploying kong components")
	deployKong(ctx, t, env, entPostgresPath, enterpriseLicenseSecretYAML, adminPasswordSecretYAML)

	t.Log("this deployment used a postgres backend, verifying that postgres migrations ran properly")
	verifyPostgres(ctx, t, env)

	t.Log("running ingress tests to verify ingress controller and proxy are functional")
	verifyIngress(ctx, t, env)

	t.Log("this deployment used enterprise kong, verifying that enterprise functionality was set up properly")
	verifyEnterprise(ctx, t, env, adminPassword)
	verifyEnterpriseWithPostgres(ctx, t, env, adminPassword)
}

// -----------------------------------------------------------------------------
// Private Functions - Test Helpers
// -----------------------------------------------------------------------------

const (
	httpBinImage = "kennethreitz/httpbin"
	ingressClass = "kong"
	namespace    = "kong"
)

func deployKong(ctx context.Context, t *testing.T, env environments.Environment, manifestPath string, additionalManifests ...string) {
	t.Log("creating a tempfile for kubeconfig")
	kubeconfig, err := generators.NewKubeConfigForRestConfig(env.Name(), env.Cluster().Config())
	require.NoError(t, err)
	kubeconfigFile, err := os.CreateTemp(os.TempDir(), "manifest-tests-kubeconfig-")
	require.NoError(t, err)
	defer os.Remove(kubeconfigFile.Name())
	defer kubeconfigFile.Close()

	t.Log("dumping kubeconfig to tempfile")
	written, err := kubeconfigFile.Write(kubeconfig)
	require.NoError(t, err)
	require.Equal(t, len(kubeconfig), written)

	t.Log("waiting for testing environment to be ready")
	require.NoError(t, <-env.WaitForReady(ctx))

	t.Log("creating the kong namespace")
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFile.Name(), "create", "namespace", namespace)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	require.NoError(t, cmd.Run(), fmt.Sprintf("STDOUT=(%s), STDERR=(%s)", stdout.String(), stderr.String()))

	t.Logf("deploying any supplemental manifests (found: %d)", len(additionalManifests))
	for _, manifest := range additionalManifests {
		stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
		cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFile.Name(), "apply", "-f", "-")
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)
		written, err := io.WriteString(stdin, manifest)
		require.NoError(t, err)
		require.Equal(t, written, len(manifest))
		require.NoError(t, stdin.Close())
		require.NoError(t, cmd.Run(), fmt.Sprintf("STDOUT=(%s), STDERR=(%s)", stdout.String(), stderr.String()))
	}

	t.Logf("deploying the %s manifest to the cluster", strings.TrimPrefix(manifestPath, "../../"))
	stdout, stderr = new(bytes.Buffer), new(bytes.Buffer)
	cmd = exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigFile.Name(), "apply", "-f", manifestPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	require.NoError(t, cmd.Run(), fmt.Sprintf("STDOUT=(%s), STDERR=(%s)", stdout.String(), stderr.String()))

	t.Log("waiting for kong to be ready")
	require.Eventually(t, func() bool {
		deployment, err := env.Cluster().Client().AppsV1().Deployments(namespace).Get(ctx, "ingress-kong", metav1.GetOptions{})
		require.NoError(t, err)
		return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
	}, kongComponentWait, time.Second)
}

func verifyIngress(ctx context.Context, t *testing.T, env environments.Environment) {
	t.Log("deploying an HTTP service to test the ingress controller and proxy")
	container := generators.NewContainer("httpbin", httpBinImage, 80)
	deployment := generators.NewDeploymentForContainer(container)
	deployment, err := env.Cluster().Client().AppsV1().Deployments(corev1.NamespaceDefault).Create(ctx, deployment, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Logf("exposing deployment %s via service", deployment.Name)
	service := generators.NewServiceForDeployment(deployment, corev1.ServiceTypeLoadBalancer)
	_, err = env.Cluster().Client().CoreV1().Services(corev1.NamespaceDefault).Create(ctx, service, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Logf("creating an ingress for service %s with ingress.class %s", service.Name, ingressClass)
	kubernetesVersion, err := env.Cluster().Version()
	require.NoError(t, err)
	ingress := generators.NewIngressForServiceWithClusterVersion(kubernetesVersion, "/httpbin", map[string]string{
		annotations.IngressClassKey: ingressClass,
		"konghq.com/strip-path":     "true",
	}, service)
	require.NoError(t, clusters.DeployIngress(ctx, env.Cluster(), corev1.NamespaceDefault, ingress))

	t.Log("finding the kong proxy service ip")
	svc, err := env.Cluster().Client().CoreV1().Services(namespace).Get(ctx, "kong-proxy", metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, svc.Status.LoadBalancer.Ingress, 1)
	proxyIP := svc.Status.LoadBalancer.Ingress[0].IP

	t.Log("waiting for routes from Ingress to be operational")
	httpc := http.Client{Timeout: time.Second * 10}
	require.Eventually(t, func() bool {
		resp, err := httpc.Get(fmt.Sprintf("http://%s/httpbin", proxyIP))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			b := new(bytes.Buffer)
			n, err := b.ReadFrom(resp.Body)
			require.NoError(t, err)
			require.True(t, n > 0)
			return strings.Contains(b.String(), "<title>httpbin.org</title>")
		}
		return false
	}, ingressWait, time.Second)
}

func verifyEnterprise(ctx context.Context, t *testing.T, env environments.Environment, adminPassword string) {
	t.Log("finding the ip address for the admin API")
	service, err := env.Cluster().Client().CoreV1().Services(namespace).Get(ctx, "kong-admin", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, len(service.Status.LoadBalancer.Ingress))
	adminIP := service.Status.LoadBalancer.Ingress[0].IP

	t.Log("building a GET request to gather admin api information")
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/", adminIP), nil)
	require.NoError(t, err)
	req.Header.Set("Kong-Admin-Token", adminPassword)

	t.Log("pulling the admin api information")
	httpc := http.Client{Timeout: time.Second * 10}
	resp, err := httpc.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("STATUS=(%s), BODY=(%s)", resp.Status, string(body)))

	t.Log("verifying the admin api version is enterprise")
	adminOutput := struct {
		Version string `json:"version"`
	}{}
	require.NoError(t, json.Unmarshal(body, &adminOutput))
	require.True(t, strings.Contains(adminOutput.Version, "enterprise-edition"))
}

func verifyEnterpriseWithPostgres(ctx context.Context, t *testing.T, env environments.Environment, adminPassword string) {
	t.Log("finding the ip address for the admin API")
	service, err := env.Cluster().Client().CoreV1().Services(namespace).Get(ctx, "kong-admin", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, len(service.Status.LoadBalancer.Ingress))
	adminIP := service.Status.LoadBalancer.Ingress[0].IP

	t.Log("building a POST request to create a new kong workspace")
	form := url.Values{"name": {"kic-e2e-tests"}}
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://%s/workspaces", adminIP), strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Kong-Admin-Token", adminPassword)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	t.Log("creating a workspace to validate enterprise functionality")
	httpc := http.Client{Timeout: time.Second * 10}
	resp, err := httpc.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, fmt.Sprintf("STATUS=(%s), BODY=(%s)", resp.Status, string(body)))
}

func verifyPostgres(ctx context.Context, t *testing.T, env environments.Environment) {
	t.Log("verifying that postgres pod was deployed and is running")
	postgresPod, err := env.Cluster().Client().CoreV1().Pods(namespace).Get(ctx, "postgres-0", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, corev1.PodRunning, postgresPod.Status.Phase)

	t.Log("verifying that all migrations ran properly")
	migrationJob, err := env.Cluster().Client().BatchV1().Jobs(namespace).Get(ctx, "kong-migrations", metav1.GetOptions{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, migrationJob.Status.Succeeded, int32(1))
}

// -----------------------------------------------------------------------------
// Private Functions - Utilities
// -----------------------------------------------------------------------------

const (
	// adminPasswordSecretName is the name of the secret which will house the admin
	// API admin password.
	adminPasswordSecretName = "kong-enterprise-superuser-password"
)

func generateAdminPasswordSecret() (string, string, error) {
	adminPassword, err := password.Generate(64, 10, 10, false, false)
	if err != nil {
		return "", "", err
	}
	adminPasswordB64 := base64.StdEncoding.EncodeToString([]byte(adminPassword))
	adminPasswordSecretYAML := fmt.Sprintf(`---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: kong
type: Opaque
data:
  password: %s`, adminPasswordSecretName, adminPasswordB64)
	return adminPassword, adminPasswordSecretYAML, nil
}