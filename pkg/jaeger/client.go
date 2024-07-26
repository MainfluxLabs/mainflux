// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jaeger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/opentracing/opentracing-go"
	jconfig "github.com/uber/jaeger-client-go/config"
)

func Init(svcName, url string, logger logger.Logger) (opentracing.Tracer, io.Closer) {
	if url == "" {
		return opentracing.NoopTracer{}, ioutil.NopCloser(nil)
	}

	tracer, closer, err := jconfig.Configuration{
		ServiceName: svcName,
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jconfig.ReporterConfig{
			LocalAgentHostPort: url,
			LogSpans:           true,
		},
	}.NewTracer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger client: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}
