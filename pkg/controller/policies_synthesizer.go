package controller

import (
	networking "k8s.io/api/networking/v1"
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
	policies, errs := PoliciesFromFolderPath(dirPath, ps.stopOnError)
	ps.errors = errs
	for _, err := range errs {
		if err.IsFatal() {
			return nil, err.Error()
		}
	}
	return policies, nil
}
