// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package uuid

import (
	"fmt"
	"strconv"
	"sync"
)

// Prefix represents the prefix used to generate UUID mocks
const Prefix = "123e4567-e89b-12d3-a456-"

var _ IDProvider = (*uuidProviderMock)(nil)

type uuidProviderMock struct {
	mu      sync.Mutex
	counter int
}

func (up *uuidProviderMock) ID() (string, error) {
	up.mu.Lock()
	defer up.mu.Unlock()

	up.counter++
	return fmt.Sprintf("%s%012d", Prefix, up.counter), nil
}

// NewMock creates "mirror" uuid provider, i.e. generated
// token will hold value provided by the caller.
func NewMock() IDProvider {
	return &uuidProviderMock{}
}

func ParseID(ID string) (id uint64) {
	var serialNum string

	if len(ID) == 36 {
		serialNum = ID[len(ID)-6:]
	}
	id, _ = strconv.ParseUint(serialNum, 10, 64)

	return
}
