// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"maps"
	mathrand "math/rand"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mfxsdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

var (
	errInvalidOrgs         = errors.New("orgs must be at least 1")
	errInvalidGroups       = errors.New("groups must be at least 1")
	errInvalidThings       = errors.New("things must be at least 1")
	errInvalidBackfillDays = errors.New("backfill days must be at least 1")
	errInvalidBatch        = errors.New("batch must be at least 1")
)

var (
	benchOrgs         int
	benchGroups       int
	benchThings       int
	benchMessages     int64
	benchBackfillDays int
	benchBatch        int
	benchRate         int
	benchMTLS         bool
)

// ECDSA P-256 keys are issued because the certs service generates them faster than RSA keys.
const (
	benchCertKeyType = "ecdsa"
	benchCertKeyBits = 256
	benchCertTTL     = "8760h"
)

// benchVariant pairs a JSON timestamp field with the transformer time format
// that parses it, so every supported format is exercised.
type benchVariant struct {
	timeField  string
	timeFormat string
	// encode returns the timestamp as a JSON value in the variant's time format.
	encode func(time.Time) string
}

var benchVariants = []benchVariant{
	{
		timeField:  "time",
		timeFormat: "unix",
		encode:     func(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) },
	},
	{
		timeField:  "ts",
		timeFormat: "unix_ms",
		encode:     func(t time.Time) string { return strconv.FormatInt(t.UnixMilli(), 10) },
	},
	{
		timeField:  "timestamp",
		timeFormat: "unix_us",
		encode:     func(t time.Time) string { return strconv.FormatInt(t.UnixMicro(), 10) },
	},
	{
		timeField:  "reading_time",
		timeFormat: "rfc3339",
		encode:     func(t time.Time) string { return strconv.Quote(t.UTC().Format(time.RFC3339)) },
	},
	{
		timeField:  "t",
		timeFormat: "iso8601",
		encode: func(t time.Time) string {
			return strconv.Quote(t.UTC().Format("2006-01-02T15:04:05"))
		},
	},
}

// benchThingTypes cycles through the valid thing types for provisioned devices.
var benchThingTypes = []string{"device", "sensor", "actuator", "controller", "gateway"}

// benchPublisher holds the credentials and message format for one device.
type benchPublisher struct {
	thingID  string
	keyType  string
	keyValue string
	variant  int
	// cert authenticates the device over mutual TLS instead of the Authorization header.
	cert *tls.Certificate
}

// NewBenchmarkCmd returns the benchmark command.
func NewBenchmarkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "benchmark <username> <password>",
		Short: "Provision entities and run an HTTP JSON message load benchmark",
		Long: `Provisions orgs, groups, profiles with transformers, and things, then
publishes backfilled JSON messages over the HTTP adapter until the configured
number of messages is sent.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			if err := validateBenchFlags(); err != nil {
				logError(err)
				return
			}

			user := mfxsdk.User{
				Email:    args[0],
				Password: args[1],
			}
			token, err := sdk.CreateToken(user)
			if err != nil {
				logError(err)
				return
			}

			publishers, err := benchProvision(token)
			if err != nil {
				logError(err)
				return
			}
			fmt.Printf("provisioned %d things across %d orgs and %d groups\n",
				len(publishers), benchOrgs, benchOrgs*benchGroups)

			benchRun(publishers)
		},
	}

	flags := cmd.Flags()
	flags.IntVar(&benchOrgs, "orgs", 3, "Number of orgs to create")
	flags.IntVar(&benchGroups, "groups", 5, "Number of groups per org")
	flags.IntVar(&benchThings, "things", 100, "Number of things to create")
	flags.Int64Var(&benchMessages, "messages", 10000, "Number of messages to send")
	flags.IntVar(&benchBackfillDays, "backfill-days", 30,
		"Spread message timestamps over the last N days")
	flags.IntVar(&benchBatch, "batch", 100, "Maximum number of messages per HTTP request")
	flags.IntVar(&benchRate, "rate", 0,
		"Approximate maximum messages per second (0 disables the limit)")
	flags.BoolVar(&benchMTLS, "mtls", false,
		"Authenticate a random half of the things with client certificates over mutual TLS")

	return cmd
}

// validateBenchFlags rejects flag values that leave no entity to provision or
// no time window to backfill.
func validateBenchFlags() error {
	switch {
	case benchOrgs < 1:
		return errInvalidOrgs
	case benchGroups < 1:
		return errInvalidGroups
	case benchThings < 1:
		return errInvalidThings
	case benchBackfillDays < 1:
		return errInvalidBackfillDays
	case benchBatch < 1:
		return errInvalidBatch
	default:
		return nil
	}
}

// benchProvision creates the orgs, groups, transformer profiles and things,
// and returns one publisher per thing.
func benchProvision(token string) ([]benchPublisher, error) {
	type slot struct {
		profileID string
		variant   int
	}
	var slots []slot

	runID := strings.ToLower(rand.Text()[:6])

	groupIdx := 0
	for o := 0; o < benchOrgs; o++ {
		org := mfxsdk.Org{Name: fmt.Sprintf("bench-%s-org-%d", runID, o)}
		orgID, err := sdk.CreateOrg(org, token)
		if err != nil {
			return nil, fmt.Errorf("create org: %w", err)
		}

		for g := 0; g < benchGroups; g++ {
			group := mfxsdk.Group{Name: fmt.Sprintf("bench-%s-group-%d-%d", runID, o, g)}
			groupID, err := sdk.CreateGroup(group, orgID, token)
			if err != nil {
				return nil, fmt.Errorf("create group: %w", err)
			}

			variant := groupIdx % len(benchVariants)
			profileID, err := sdk.CreateProfile(mfxsdk.Profile{
				Name:   fmt.Sprintf("bench-%s-profile-%d-%d", runID, o, g),
				Config: benchProfileConfig(variant),
			}, groupID, token)
			if err != nil {
				return nil, fmt.Errorf("create profile: %w", err)
			}

			slots = append(slots, slot{
				profileID: profileID,
				variant:   variant,
			})
			groupIdx++
		}
	}

	publishers := make([]benchPublisher, 0, benchThings)
	for i := 0; i < benchThings; i++ {
		s := slots[i%len(slots)]

		thingID, err := sdk.CreateThing(mfxsdk.Thing{
			Name: fmt.Sprintf("bench-%s-device-%03d", runID, i),
			Type: benchThingTypes[i%len(benchThingTypes)],
		}, s.profileID, token)
		if err != nil {
			return nil, fmt.Errorf("create thing: %w", err)
		}

		thing, err := sdk.GetThing(thingID, token)
		if err != nil {
			return nil, fmt.Errorf("get thing: %w", err)
		}

		p := benchPublisher{
			thingID: thingID,
			variant: s.variant,
		}
		// Odd indexes use an external key so that half of the things, rounded down, use one.
		if i%2 == 1 {
			extKey := rand.Text()
			if err := sdk.UpdateExternalThingKey(extKey, thingID, token); err != nil {
				return nil, fmt.Errorf("set external key: %w", err)
			}
			p.keyType = "External "
			p.keyValue = extKey
		} else {
			p.keyType = "Thing "
			p.keyValue = thing.Key
		}
		publishers = append(publishers, p)
	}

	if benchMTLS {
		for _, i := range mathrand.Perm(len(publishers))[:len(publishers)/2] {
			cert, err := benchIssueCert(publishers[i].thingID, token)
			if err != nil {
				return nil, err
			}
			publishers[i].cert = &cert
		}
	}

	return publishers, nil
}

// benchIssueCert issues a client certificate for the thing through the certs service.
func benchIssueCert(thingID, token string) (tls.Certificate, error) {
	c, err := sdk.IssueCert(thingID, benchCertKeyBits, benchCertKeyType, benchCertTTL, token)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("issue certificate: %w", err)
	}

	cert, err := tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse certificate: %w", err)
	}

	return cert, nil
}

// benchProfileConfig builds a JSON profile config whose transformer parses the
// variant's timestamp field.
func benchProfileConfig(variant int) map[string]any {
	v := benchVariants[variant]
	return map[string]any{
		"content_type":  "application/json",
		"write_enabled": true,
		"transformer": map[string]any{
			"time_field":    v.timeField,
			"time_format":   v.timeFormat,
			"time_location": "UTC",
		},
	}
}

// benchRun floods the HTTP adapter with backfilled JSON messages until the
// configured number of messages is sent.
func benchRun(publishers []benchPublisher) {
	backfill := time.Duration(benchBackfillDays) * 24 * time.Hour

	var claimed int64
	var stats [benchAuthCount]benchStats
	var total benchStats
	for _, p := range publishers {
		stats[p.auth()].devices++
	}
	total.devices = len(publishers)
	failures := benchFailures{counts: map[string]int64{}}

	start := time.Now()

	// limiter throttles the total message rate so the writer and NATS are not overrun.
	var limiter *rate.Limiter
	if benchRate > 0 {
		limiter = rate.NewLimiter(rate.Limit(benchRate), benchBatch)
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				sent := total.sent.Load()
				elapsed := time.Since(start).Seconds()
				fmt.Printf("requests=%d sent=%d failed=%d data=%.2fMB rate=%.0f msg/s\n",
					total.requests.Load(), sent, total.failed.Load(),
					float64(total.bytes.Load())/1e6, float64(sent)/elapsed)
			}
		}
	}()

	// Each device publishes from its own goroutine, so concurrency follows the device count.
	var wg sync.WaitGroup
	for i, p := range publishers {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rng := mathrand.New(mathrand.NewSource(seed))
			client := benchClient(p)
			st := &stats[p.auth()]
			for {
				// Requests carry a random number of messages so that both single object
				// and JSON array payloads are exercised.
				size := int64(1 + rng.Intn(benchBatch))
				// Each batch is claimed before it is sent so that exactly benchMessages are sent.
				first := atomic.AddInt64(&claimed, size) - size
				if first >= benchMessages {
					return
				}
				n := min(size, benchMessages-first)

				if limiter != nil {
					// WaitN fails only when n exceeds the burst, which equals benchBatch.
					if err := limiter.WaitN(context.Background(), int(n)); err != nil {
						return
					}
				}
				body := benchBatchBody(p.variant, int(n), backfill, rng)

				bytes := int64(len(body))
				cause := benchPublish(client, p, body)
				st.record(n, bytes, cause)
				total.record(n, bytes, cause)
				if cause != "" {
					failures.add(cause)
				}
			}
		}(time.Now().UnixNano() + int64(i))
	}

	wg.Wait()
	close(done)

	benchReport(&stats, &total, &failures, time.Since(start))
}

// benchReport prints the provisioning summary, the counters per authentication type and the
// failed requests per cause.
func benchReport(stats *[benchAuthCount]benchStats, total *benchStats, failures *benchFailures,
	elapsed time.Duration) {
	line := strings.Repeat("─", 62)
	rule := "  " + strings.Repeat("─", 60)
	header := "  %-14s %8s %10s %12s %12s\n"
	row := "  %-14s %8d %10d %12d %12d\n"

	fmt.Printf("\n%s\n", line)
	fmt.Println("  Mainflux HTTP Benchmark — Results")
	fmt.Println(line)
	fmt.Printf("  orgs / groups / devices     %d / %d / %d\n",
		benchOrgs, benchOrgs*benchGroups, total.devices)
	fmt.Printf("  transformer formats         %d\n", len(benchVariants))
	fmt.Printf("  backfill window             %d days\n", benchBackfillDays)
	fmt.Printf("  max batch                   %d messages per request\n", benchBatch)
	fmt.Println(rule)

	fmt.Printf(header, "auth", "devices", "requests", "msgs sent", "msgs failed")
	for a := range stats {
		s := &stats[a]
		if s.devices == 0 {
			continue
		}
		fmt.Printf(row, benchAuthNames[a], s.devices, s.requests.Load(), s.sent.Load(),
			s.failed.Load())
	}
	fmt.Printf(row, "total", total.devices, total.requests.Load(), total.sent.Load(),
		total.failed.Load())
	fmt.Println(rule)

	if len(failures.counts) == 0 {
		fmt.Println("  failed requests             none")
	} else {
		fmt.Println("  failed requests by cause")
		for _, cause := range slices.Sorted(maps.Keys(failures.counts)) {
			fmt.Printf("    %-26s%d\n", cause, failures.counts[cause])
		}
	}
	fmt.Println(rule)

	secs := elapsed.Seconds()
	fmt.Printf("  data sent                   %.2f MB\n", float64(total.bytes.Load())/1e6)
	fmt.Printf("  duration                    %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  throughput                  %.0f msg/s, %.0f req/s\n",
		float64(total.sent.Load())/secs, float64(total.requests.Load())/secs)
	fmt.Println(line)
}

// benchAuth identifies how a device authenticates its requests.
type benchAuth int

const (
	benchAuthThing benchAuth = iota
	benchAuthExternal
	benchAuthMTLS
	benchAuthCount
)

var benchAuthNames = [benchAuthCount]string{"thing key", "external key", "mTLS"}

// auth returns how the publisher authenticates its requests.
func (p benchPublisher) auth() benchAuth {
	switch {
	case p.cert != nil:
		return benchAuthMTLS
	case p.keyType == "External ":
		return benchAuthExternal
	default:
		return benchAuthThing
	}
}

// benchStats holds the live counters of a set of devices.
type benchStats struct {
	devices  int
	requests atomic.Int64
	sent     atomic.Int64
	failed   atomic.Int64
	bytes    atomic.Int64
}

// record counts one request of n messages, which failed when cause is not empty.
func (s *benchStats) record(n, bytes int64, cause string) {
	s.requests.Add(1)
	if cause != "" {
		s.failed.Add(n)
		return
	}
	s.sent.Add(n)
	s.bytes.Add(bytes)
}

// benchFailures counts failed requests per cause.
type benchFailures struct {
	mu     sync.Mutex
	counts map[string]int64
}

func (bf *benchFailures) add(cause string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	bf.counts[cause]++
}

// benchClient returns an HTTP client that presents the publisher's client certificate, if any.
func benchClient(p benchPublisher) *http.Client {
	// The SDK applies the same certificate verification mode.
	tlsConf := &tls.Config{
		InsecureSkipVerify: !TLSVerification,
	}
	if p.cert != nil {
		tlsConf.Certificates = []tls.Certificate{*p.cert}
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 1,
			IdleConnTimeout:     60 * time.Second,
			TLSClientConfig:     tlsConf,
		},
	}
}

// benchPublish posts one request and returns its failure cause, or an empty string on success.
func benchPublish(client *http.Client, p benchPublisher, body string) string {
	url := HTTPAdapterURL + "/messages"
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "invalid request"
	}
	// The proxy takes the thing key from the client certificate and rejects a differing header.
	if p.cert == nil {
		req.Header.Set("Authorization", p.keyType+p.keyValue)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "connection error"
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return ""
}

// benchBatchBody builds the request body for one publisher, a JSON object for a
// single message and a JSON array otherwise, with a backfilled timestamp per message.
func benchBatchBody(variant, n int, backfill time.Duration, rng *mathrand.Rand) string {
	if n == 1 {
		ts := time.Now().Add(-time.Duration(rng.Int63n(int64(backfill))))
		return benchItem(variant, ts, rng)
	}

	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		ts := time.Now().Add(-time.Duration(rng.Int63n(int64(backfill))))
		b.WriteString(benchItem(variant, ts, rng))
	}
	b.WriteByte(']')
	return b.String()
}

// benchItem builds a multi-field JSON message carrying the variant's timestamp
// field in the matching format.
func benchItem(variant int, ts time.Time, rng *mathrand.Rand) string {
	format := `"temperature":%.2f,"humidity":%d,"pressure":%d,"co2":%d,` +
		`"voltage":%.3f,"current":%.3f,"battery":%d,"rssi":%d,"status":"ok"`
	fields := fmt.Sprintf(format,
		15+rng.Float64()*20, 30+rng.Intn(60), 980+rng.Intn(60), 400+rng.Intn(1200),
		3+rng.Float64(), rng.Float64()*2, rng.Intn(100), -40-rng.Intn(60))

	v := benchVariants[variant]
	return fmt.Sprintf(`{"%s":%s,%s}`, v.timeField, v.encode(ts), fields)
}
