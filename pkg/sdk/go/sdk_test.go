// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	limit  uint64 = 5
	offset uint64 = 0
	total  uint64 = 200
)

func createError(e error, statusCode int) error {
	httpStatus := fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
	return errors.Wrap(e, errors.New(httpStatus))
}
