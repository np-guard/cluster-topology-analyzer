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
	return ps
}

func (ps *PoliciesSynthesizer) Errors() []FileProcessingError {
	return ps.errors
}

func (ps *PoliciesSynthesizer) PoliciesFromFolderPath(dirPath string) ([]*networking.NetworkPolicy, error) {
	activeLogger = ps.logger

	emptyStr := ""
	args := common.InArgs{}
	args.DirPath = &dirPath
	args.CommitID = &emptyStr
	args.GitBranch = &emptyStr
	args.GitURL = &emptyStr

	connections, errs := extractConnections(args, ps.stopOnError)
	policies := []*networking.NetworkPolicy{}
	if !stopProcessing(ps.stopOnError, errs) {
		policies = synthNetpols(connections)
	}

	ps.errors = errs
	for idx := range errs {
		if errs[idx].IsFatal() {
			return nil, errs[idx].Error()
		}
	}

	return policies, nil
}
