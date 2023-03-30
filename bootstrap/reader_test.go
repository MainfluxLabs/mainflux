// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/bootstrap"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readChan struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type readResp struct {
	ThingID    string     `json:"thing_id"`
	ThingKey   string     `json:"thing_key"`
	Channels   []readChan `json:"channels"`
	Content    string     `json:"content,omitempty"`
	ClientCert string     `json:"client_cert,omitempty"`
	ClientKey  string     `json:"client_key,omitempty"`
	CACert     string     `json:"ca_cert,omitempty"`
}

func dec(in []byte) ([]byte, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	if len(in) < aes.BlockSize {
		return nil, errors.ErrMalformedEntity
	}
	iv := in[:aes.BlockSize]
	in = in[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(in, in)
	return in, nil
}

func TestReadConfig(t *testing.T) {
	cfg := bootstrap.Config{
		ThingID:    "mf_id",
		ClientCert: "client_cert",
		ClientKey:  "client_key",
		CACert:     "ca_cert",
		ThingKey:   "mf_key",
		Channels: []bootstrap.Channel{
			bootstrap.Channel{
				ID:       "mf_id",
				Name:     "mf_name",
				Metadata: map[string]interface{}{"key": "value}"},
			},
		},
		Content: "content",
	}
	ret := readResp{
		ThingID:  "mf_id",
		ThingKey: "mf_key",
		Channels: []readChan{
			{
				ID:       "mf_id",
				Name:     "mf_name",
				Metadata: map[string]interface{}{"key": "value}"},
			},
		},
		Content:    "content",
		ClientCert: "client_cert",
		ClientKey:  "client_key",
		CACert:     "ca_cert",
	}

	bin, err := json.Marshal(ret)
	require.Nil(t, err, fmt.Sprintf("Marshalling expected to succeed: %s.\n", err))

	reader := bootstrap.NewConfigReader(encKey)
	cases := []struct {
		desc   string
		config bootstrap.Config
		enc    []byte
		secret bool
		err    error
	}{
		{
			desc:   "read a config",
			config: cfg,
			enc:    bin,
			secret: false,
		},
		{
			desc:   "read encrypted config",
			config: cfg,
			enc:    bin,
			secret: true,
		},
	}

	for _, tc := range cases {
		res, err := reader.ReadConfig(tc.config, tc.secret)
		require.Nil(t, err, fmt.Sprintf("Reading config to succeed: %s.\n", err))

		if tc.secret {
			d, err := dec(res.([]byte))
			require.Nil(t, err, fmt.Sprintf("Decrypting expected to succeed: %s.\n", err))
			assert.Equal(t, tc.enc, d, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.enc, d))
			continue
		}
		b, err := json.Marshal(res)
		require.Nil(t, err, fmt.Sprintf("Marshalling expected to succeed: %s.\n", err))
		assert.Equal(t, tc.enc, b, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.enc, b))
		resp, ok := res.(mainflux.Response)
		require.True(t, ok, "If not encrypted, reader should return response.")
		assert.False(t, resp.Empty(), fmt.Sprintf("Response should not be empty %s.", err))
		assert.Equal(t, http.StatusOK, resp.Code(), "Default config response code should be 200.")
	}
}
