// The controller package of cluster-topology-analyzer discovers the connectivity of a Kubernetes application
// by analyzing its YAML manifests and looking for network addresses that match. It can output a set of
// discovered connections or even Kubernetes NetworkPolicies to allow only these connections. For more
// information, see https://github.com/np-guard/cluster-topology-analyzer.
package controller

import (
	"io/fs"
	"path/filepath"

	networking "k8s.io/api/networking/v1"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

const (
	DefaultDNSPort = 53
)

// Walk function is a function for recursively scanning a directory, in the spirit of Go's native filepath.WalkDir()
// See https://pkg.go.dev/path/filepath#WalkDir for full description on how such a function should work
type WalkFunction func(root string, fn fs.WalkDirFunc) error

// A PoliciesSynthesizer provides API to recursively scan a directory for Kubernetes resources
// and extract the required connectivity between the workloads of the K8s application managed in this directory.
// It is possible to get either a slice with all the discovered connections or a slice with K8s NetworkPolicies
// that allow only the discovered connections and nothing more.
type PoliciesSynthesizer struct {
	logger      Logger
	stopOnError bool
	walkFn      WalkFunction
	dnsPort     int

	errors []FileProcessingError
}

// PoliciesSynthesizerOption is the type for specifying options for PoliciesSynthesizer,
// using Golang's Options Pattern (https://golang.cafe/blog/golang-functional-options-pattern.html).
type PoliciesSynthesizerOption func(*PoliciesSynthesizer)

// WithWalkFn is a functional option, allowing user to provide their own dir-scanning function.
// The function will be used when searching for YAML files; it must have the same signature as filepath.WalkDir.
func WithWalkFn(walkFn WalkFunction) PoliciesSynthesizerOption {
	return func(p *PoliciesSynthesizer) {
		p.walkFn = walkFn
	}
}

// WithLogger is a functional option which sets the logger for a PoliciesSynthesizer to use.
// The provided logger must conform with the package's Logger interface.
func WithLogger(logger Logger) PoliciesSynthesizerOption {
	return func(p *PoliciesSynthesizer) {
		p.logger = logger
	}
}

// WithStopOnError is a functional option which directs PoliciesSynthesizer to stop any processing after the
// first severe error.
func WithStopOnError() PoliciesSynthesizerOption {
	return func(p *PoliciesSynthesizer) {
		p.stopOnError = true
	}
}

func WithDNSPort(dnsPort int) PoliciesSynthesizerOption {
	return func(p *PoliciesSynthesizer) {
		p.dnsPort = dnsPort
	}
}

// NewPoliciesSynthesizer creates a new instance of PoliciesSynthesizer, and applies the provided functional options.
func NewPoliciesSynthesizer(options ...PoliciesSynthesizerOption) *PoliciesSynthesizer {
	// object with default behavior options
	ps := &PoliciesSynthesizer{
		logger:      NewDefaultLogger(),
		stopOnError: false,
		walkFn:      filepath.WalkDir,
		dnsPort:     DefaultDNSPort,
		errors:      []FileProcessingError{},
	}
	for _, o := range options {
		o(ps)
	}
	return ps
}

// Errors returns a slice of FileProcessingError with all warnings and errors encountered during processing.
func (ps *PoliciesSynthesizer) Errors() []FileProcessingError {
	return ps.errors
}

// PoliciesFromFolderPath returns a slice of Kubernetes NetworkPolicies that allow only the connections discovered
// while processing K8s resources under the provided directory or one of its subdirectories (recursively).
func (ps *PoliciesSynthesizer) PoliciesFromFolderPath(dirPath string) ([]*networking.NetworkPolicy, error) {
	return ps.PoliciesFromFolderPaths([]string{dirPath})
}

// PoliciesFromFolderPath returns a slice of Kubernetes NetworkPolicies that allow only the connections discovered
// while processing K8s resources under the provided directories or one of their subdirectories (recursively).
func (ps *PoliciesSynthesizer) PoliciesFromFolderPaths(dirPaths []string) ([]*networking.NetworkPolicy, error) {
	resources, connections, errs := ps.extractConnections(dirPaths)
	policies := []*networking.NetworkPolicy{}
	if !stopProcessing(ps.stopOnError, errs) {
		policies = ps.synthNetpols(resources, connections)
	}

	ps.errors = errs
	if err := hasFatalError(errs); err != nil {
		return nil, err
	}

	return policies, nil
}

// ConnectionsFromFolderPath returns a slice of Connections, listing the connections discovered
// while processing K8s resources under the provided directory or one of its subdirectories (recursively).
func (ps *PoliciesSynthesizer) ConnectionsFromFolderPath(dirPath string) ([]*common.Connections, error) {
	return ps.ConnectionsFromFolderPaths([]string{dirPath})
}

// ConnectionsFromFolderPath returns a slice of Connections, listing the connections discovered
// while processing K8s resources under the provided directories or one of their subdirectories (recursively).
func (ps *PoliciesSynthesizer) ConnectionsFromFolderPaths(dirPaths []string) ([]*common.Connections, error) {
	_, connections, errs := ps.extractConnections(dirPaths)
	ps.errors = errs
	if err := hasFatalError(errs); err != nil {
		return nil, err
	}

	return connections, nil
}

// Scans the given directory for YAMLs with k8s resources and extracts required connections between workloads
func (ps *PoliciesSynthesizer) extractConnections(dirPaths []string) ([]common.Resource, []*common.Connections, []FileProcessingError) {
	// 1. Get all relevant resources from the repo
	resFinder := resourceFinder{logger: ps.logger, stopOn1stErr: ps.stopOnError, walkFn: ps.walkFn}
	resources := []common.Resource{}
	links := []common.Service{}
	configmaps := []common.CfgMap{}
	fileErrors := []FileProcessingError{}
	for _, dirPath := range dirPaths {
		r, l, c, errs := resFinder.getRelevantK8sResources(dirPath)
		resources = append(resources, r...)
		links = append(links, l...)
		configmaps = append(configmaps, c...)
		fileErrors = append(fileErrors, errs...)
		if stopProcessing(ps.stopOnError, errs) {
			return nil, nil, fileErrors
		}
	}
	if len(resources) == 0 {
		fileErrors = appendAndLogNewError(fileErrors, noK8sResourcesFound(), ps.logger)
		return []common.Resource{}, []*common.Connections{}, fileErrors
	}

	// 2. Inline configmaps values as workload envs
	errs := inlineConfigMapRefsAsEnvs(resources, configmaps, ps.logger)
	fileErrors = append(fileErrors, errs...)
	if stopProcessing(ps.stopOnError, fileErrors) {
		return nil, nil, fileErrors
	}

	// 3. Discover all connections between resources
	connections := discoverConnections(resources, links, ps.logger)
	return resources, connections, fileErrors
}

func hasFatalError(errs []FileProcessingError) error {
	for idx := range errs {
		if errs[idx].IsFatal() {
			return errs[idx].Error()
		}
	}
	return nil
}
