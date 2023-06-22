package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

func TestNetworkAddressValue(t *testing.T) {

	type strBoolPair struct {
		str string
		b   bool
	}

	valuesToCheck := map[string]strBoolPair{
		"svc":                           {"svc", true},
		"svc:500":                       {"svc:500", true},
		"http://svc:500":                {"svc:500", true},
		"fttps://svc:500/something#abc": {"svc:500", true},
		strings.Repeat("abc", 500):      {"", false},
		"not%a*url":                     {"", false},
		"123":                           {"", false},
	}

	for val, expectedAnswer := range valuesToCheck {
		strRes, boolRes := NetworkAddressValue(val)
		require.Equal(t, expectedAnswer.b, boolRes)
		require.Equal(t, expectedAnswer.str, strRes)
	}
}

func TestScanningSvc(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"k8s_guestbook", "frontend-service.yaml"})
	require.Nil(t, err)
	res, err := ScanK8sServiceObject(resourceBuf)
	require.Nil(t, err)
	require.Equal(t, "frontend", res.Resource.Name)
	require.Len(t, res.Resource.Selectors, 2)
	require.Len(t, res.Resource.Network, 1)
	require.Equal(t, 80, res.Resource.Network[0].Port)
}

func TestScanningDeploymentWithArgs(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"sockshop", "manifests", "01-carts-dep.yaml"})
	require.Nil(t, err)
	res, err := ScanK8sWorkloadObject("Deployment", resourceBuf)
	require.Nil(t, err)
	require.Equal(t, "carts", res.Resource.Name)
	require.Len(t, res.Resource.NetworkAddrs, 1)
	require.Equal(t, "carts-db:27017", res.Resource.NetworkAddrs[0])
	require.Len(t, res.Resource.Labels, 1)
	require.Equal(t, "carts", res.Resource.Labels["name"])
}

func TestScanningDeploymentWithEnvs(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"k8s_guestbook", "frontend-deployment.yaml"})
	require.Nil(t, err)
	res, err := ScanK8sWorkloadObject("Deployment", resourceBuf)
	require.Nil(t, err)
	require.Equal(t, "frontend", res.Resource.Name)
	require.Len(t, res.Resource.NetworkAddrs, 4)
	require.Len(t, res.Resource.Labels, 2)
}

func TestScanningDeploymentWithConfigMapRef(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"acs-security-demos", "frontend", "webapp", "deployment.yaml"})
	require.Nil(t, err)
	res, err := ScanK8sWorkloadObject("Deployment", resourceBuf)
	require.Nil(t, err)
	require.Equal(t, "webapp", res.Resource.Name)
	require.Empty(t, res.Resource.NetworkAddrs) // extracting network addresses from configmaps happens later
	require.Len(t, res.Resource.Labels, 1)
}

func TestScanningReplicaSet(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"k8s_guestbook", "redis-leader-deployment.yaml"})
	require.Nil(t, err)
	res, err := ScanK8sWorkloadObject("ReplicaSet", resourceBuf)
	require.Nil(t, err)
	require.Equal(t, "redis-leader", res.Resource.Name)
	require.Len(t, res.Resource.NetworkAddrs, 0)
	require.Len(t, res.Resource.Labels, 3)
}

func TestScanningConfigMap(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"qotd", "qotd_usecase.yaml"})
	require.Nil(t, err)
	res, err := ScanK8sConfigmapObject(resourceBuf)
	require.Nil(t, err)
	require.Equal(t, res.FullName, "qotd-load/qotd-usecase-library")
	require.Len(t, res.Data, 5)
}

func TestScanningIngress(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"bookinfo", "bookinfo-ingress.yaml"})
	require.Nil(t, err)
	toExpose := common.ServicesToExpose{}
	err = ScanIngressObject(resourceBuf, toExpose)
	require.Nil(t, err)
	require.Len(t, toExpose, 1)
}

func TestScanningRoute(t *testing.T) {
	resourceBuf, err := loadResourceAsByteArray([]string{"acs-security-demos", "frontend", "webapp", "route.yaml"})
	require.Nil(t, err)
	toExpose := common.ServicesToExpose{}
	err = ScanOCRouteObject(resourceBuf, toExpose)
	require.Nil(t, err)
	require.Len(t, toExpose, 1)
}

func loadResourceAsByteArray(resourceDirs []string) ([]byte, error) {
	currentDir, _ := os.Getwd()
	resourceRelPath := filepath.Join(resourceDirs...)
	resourcePath := filepath.Join(currentDir, "..", "..", "tests", resourceRelPath)
	return os.ReadFile(resourcePath)
}
