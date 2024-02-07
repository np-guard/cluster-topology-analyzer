package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/resource"

	"github.com/np-guard/netpol-analyzer/pkg/netpol/manifests/fsscanner"
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
		strRes, boolRes := networkAddressFromStr(val)
		require.Equal(t, expectedAnswer.b, boolRes)
		require.Equal(t, expectedAnswer.str, strRes)
	}
}

func TestScanningSvc(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"k8s_guestbook", "frontend-service.yaml"}, 0)
	require.Nil(t, err)
	res, err := k8sServiceFromInfo(resourceInfo)
	require.Nil(t, err)
	require.Equal(t, "frontend", res.Resource.Name)
	require.Len(t, res.Resource.Selectors, 2)
	require.Len(t, res.Resource.Network, 1)
	require.Equal(t, 80, res.Resource.Network[0].Port)
}

func TestScanningDeploymentWithArgs(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"sockshop", "manifests", "01-carts-dep.yaml"}, 0)
	require.Nil(t, err)
	res, err := k8sWorkloadObjectFromInfo(resourceInfo)
	require.Nil(t, err)
	require.Equal(t, "carts", res.Resource.Name)
	require.Len(t, res.Resource.NetworkAddrs, 1)
	require.Equal(t, "carts-db:27017", res.Resource.NetworkAddrs[0])
	require.Len(t, res.Resource.Labels, 1)
	require.Equal(t, "carts", res.Resource.Labels["name"])
}

func TestScanningDeploymentWithEnvs(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"k8s_guestbook", "frontend-deployment.yaml"}, 0)
	require.Nil(t, err)
	res, err := k8sWorkloadObjectFromInfo(resourceInfo)
	require.Nil(t, err)
	require.Equal(t, "frontend", res.Resource.Name)
	require.Len(t, res.Resource.NetworkAddrs, 4)
	require.Len(t, res.Resource.Labels, 2)
}

func TestScanningDeploymentWithConfigMapRef(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"acs-security-demos", "frontend", "webapp", "deployment.yaml"}, 0)
	require.Nil(t, err)
	res, err := k8sWorkloadObjectFromInfo(resourceInfo)
	require.Nil(t, err)
	require.Equal(t, "webapp", res.Resource.Name)
	require.Len(t, res.Resource.ConfigMapRefs, 1)
	require.Empty(t, res.Resource.NetworkAddrs) // extracting network addresses from configmaps happens later
	require.Len(t, res.Resource.Labels, 1)
}

func TestScanningReplicaSet(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"k8s_guestbook", "redis-leader-deployment.yaml"}, 0)
	require.Nil(t, err)
	res, err := k8sWorkloadObjectFromInfo(resourceInfo)
	require.Nil(t, err)
	require.Equal(t, "redis-leader", res.Resource.Name)
	require.Len(t, res.Resource.NetworkAddrs, 0)
	require.Len(t, res.Resource.Labels, 3)
}

func TestScanningConfigMap(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"qotd", "qotd_usecase.yaml"}, 0)
	require.Nil(t, err)
	res, err := k8sConfigmapFromInfo(resourceInfo)
	require.Nil(t, err)
	require.Equal(t, res.FullName, "qotd-load/qotd-usecase-library")
	require.Len(t, res.Data, 5)
}

func TestScanningIngress(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"bookinfo", "bookinfo-ingress.yaml"}, 0)
	require.Nil(t, err)
	toExpose := servicesToExpose{}
	err = k8sIngressFromInfo(resourceInfo, toExpose)
	require.Nil(t, err)
	require.Len(t, toExpose, 1)
}

func TestScanningRoute(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"acs-security-demos", "frontend", "webapp", "route.yaml"}, 0)
	require.Nil(t, err)
	toExpose := servicesToExpose{}
	err = ocRouteFromInfo(resourceInfo, toExpose)
	require.Nil(t, err)
	require.Len(t, toExpose, 1)
}

func TestScanningCronJob(t *testing.T) {
	resourceInfo, err := loadResourceAsInfo([]string{"openshift", "openshift-operator-lifecycle-manager-resources.yaml"}, 7)
	require.Nil(t, err)
	res, err := k8sWorkloadObjectFromInfo(resourceInfo)
	require.Nil(t, err)
	require.Equal(t, "collect-profiles", res.Resource.Name)
	require.Equal(t, cronJob, res.Resource.Kind)
	require.Len(t, res.Resource.NetworkAddrs, 1)
	require.Len(t, res.Resource.Labels, 0)
}

func loadResourceAsInfo(resourceDirs []string, infoIndex int) (*resource.Info, error) {
	currentDir, _ := os.Getwd()
	resourceRelPath := filepath.Join(resourceDirs...)
	resourcePath := filepath.Join(currentDir, "..", "..", "tests", resourceRelPath)

	infos, errs := fsscanner.GetResourceInfosFromDirPath([]string{resourcePath}, true, true)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	if len(infos) <= infoIndex {
		return nil, fmt.Errorf("Info  number %d was required, but only %d Infos were read", infoIndex, len(infos))
	}

	return infos[infoIndex], nil
}
