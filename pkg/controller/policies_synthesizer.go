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

// Walk function is a function for recursively scanning a directory, in the spirit of Go's native filepath.WalkDir()
// See https://pkg.go.dev/path/filepath#WalkDir for full description on how such file should work
type WalkFunction func(root string, fn fs.WalkDirFunc) error

// A PoliciesSynthesizer provides API to recursively scan a directory for Kubernetes resources
// and extract the required connectivity between the workloads of the K8s application managed in this directory.
// It is possible to get either a slice with all the discovered connections or a slice with K8s NetworkPolicies
// that allow only the discovered connections and nothing more.
type PoliciesSynthesizer struct {
	logger      Logger
	stopOnError bool
	walkFn      WalkFunction

	errors []FileProcessingError
}

// PoliciesSynthesizerOption is the type for specifying options for PoliciesSynthesizer,
// using Golang's Options Pattern (https://golang.cafe/blog/golang-functional-options-pattern.html).
type PoliciesSynthesizerOption func(*PoliciesSynthesizer)

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

// NewPoliciesSynthesizer creates a new instance of PoliciesSynthesizer, and applies the provided functional options.
func NewPoliciesSynthesizer(options ...PoliciesSynthesizerOption) *PoliciesSynthesizer {
	// object with default behavior options
	ps := &PoliciesSynthesizer{
		logger:      NewDefaultLogger(),
		stopOnError: false,
		walkFn:      filepath.WalkDir,
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
	resources, connections, errs := ps.extractConnections(dirPath)
	policies := []*networking.NetworkPolicy{}
	if !stopProcessing(ps.stopOnError, errs) {
		policies = synthNetpols(resources, connections)
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
	_, connections, errs := ps.extractConnections(dirPath)
	ps.errors = errs
	if err := hasFatalError(errs); err != nil {
		return nil, err
	}

	return connections, nil
}

// Scans the given directory for YAMLs with k8s resources and extracts required connections between workloads
func (ps *PoliciesSynthesizer) extractConnections(dirPath string) ([]common.Resource, []*common.Connections, []FileProcessingError) {
	// 1. Get all relevant resources from the repo
	resFinder := resourceFinder{logger: ps.logger, stopOn1stErr: ps.stopOnError, walkFn: ps.walkFn}
	dObjs, fileErrors := resFinder.getRelevantK8sResources(dirPath)
	if stopProcessing(ps.stopOnError, fileErrors) {
		return nil, nil, fileErrors
	}
	if len(dObjs) == 0 {
		fileErrors = appendAndLogNewError(fileErrors, noK8sResourcesFound(), ps.logger)
		return []common.Resource{}, []*common.Connections{}, fileErrors
	}

	// 2. Parse them into internal structs
	resParser := resourceParser{logger: ps.logger}
	resources, links, parseErrors := resParser.parseResources(dObjs)
	fileErrors = append(fileErrors, parseErrors...)
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
