package controller

import (
	networking "k8s.io/api/networking/v1"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

type PoliciesSynthesizer struct {
	logger      Logger
	stopOnError bool

	errors []FileProcessingError
}

type PoliciesSynthesizerOption func(*PoliciesSynthesizer)

func WithLogger(logger Logger) PoliciesSynthesizerOption {
	return func(p *PoliciesSynthesizer) {
		p.logger = logger
	}
}

func WithStopOnError() PoliciesSynthesizerOption {
	return func(p *PoliciesSynthesizer) {
		p.stopOnError = true
	}
}

func NewPoliciesSynthesizer(options ...PoliciesSynthesizerOption) *PoliciesSynthesizer {
	// object with default behavior options
	ps := &PoliciesSynthesizer{
		logger:      NewDefaultLogger(),
		stopOnError: false,
		errors:      []FileProcessingError{},
	}
	for _, o := range options {
		o(ps)
	}
	activeLogger = ps.logger
	return ps
}

func (ps *PoliciesSynthesizer) Errors() []FileProcessingError {
	return ps.errors
}

func (ps *PoliciesSynthesizer) PoliciesFromFolderPath(dirPath string) ([]*networking.NetworkPolicy, error) {
	connections, errs := extractConnections(dirPath, ps.stopOnError)
	policies := []*networking.NetworkPolicy{}
	if !stopProcessing(ps.stopOnError, errs) {
		policies = synthNetpols(connections)
	}

	ps.errors = errs
	if err := hasFatalError(errs); err != nil {
		return nil, err
	}

	return policies, nil
}

func (ps *PoliciesSynthesizer) ConnectionsFromFolderPath(dirPath string) ([]*common.Connections, error) {
	connections, errs := extractConnections(dirPath, ps.stopOnError)
	ps.errors = errs
	if err := hasFatalError(errs); err != nil {
		return nil, err
	}

	return connections, nil
}

func hasFatalError(errs []FileProcessingError) error {
	for idx := range errs {
		if errs[idx].IsFatal() {
			return errs[idx].Error()
		}
	}
	return nil
}
