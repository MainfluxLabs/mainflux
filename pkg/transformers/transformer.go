// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package transformers

import protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"

// Transformer specifies API form Message transformer.
type Transformer interface {
	// Transform Mainflux message to any other format.
	Transform(msg protomfx.Message) (interface{}, error)
}
