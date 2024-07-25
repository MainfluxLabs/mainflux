// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package clients

type Config struct {
	ClientName string
	ClientTLS  bool
	CaCerts    string
	URL        string
}
