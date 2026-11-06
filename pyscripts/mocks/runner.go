// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pyscripts"
)

var _ pyscripts.Runner = (*Runner)(nil)

type Runner struct {
	mu          sync.Mutex
	Result      pyscripts.Result
	RunErr      error
	Invalid     map[string]string
	ValidateErr error
	Closed      bool
}

func NewRunner() *Runner {
	return &Runner{Invalid: make(map[string]string)}
}

func (r *Runner) Run(_ context.Context, _ string, _ pyscripts.Input) (pyscripts.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.RunErr != nil {
		return pyscripts.Result{}, r.RunErr
	}

	return r.Result, nil
}

func (r *Runner) Validate(_ context.Context, code string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ValidateErr != nil {
		return "", r.ValidateErr
	}

	return r.Invalid[code], nil
}

func (r *Runner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Closed = true
	return nil
}
