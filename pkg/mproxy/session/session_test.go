package session_test

import (
	"crypto/x509"
	"io"
	"net"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mproxy/session"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/stretchr/testify/assert"
)

const readTimeout = 500 * time.Millisecond

var errUnauthorized = errors.New("unauthorized")

// recordingHandler is a session.Handler that reports, over channels, which hooks
// the proxy invoked. Tests assert on those channels rather than on shared fields,
// so they never race the Session goroutine.
type recordingHandler struct {
	authPublishErr error
	published      chan string         // one topic per Publish hook
	disconnected   chan session.Client // the client state seen at Disconnect
}

func newRecordingHandler(authPublishErr error) *recordingHandler {
	return &recordingHandler{
		authPublishErr: authPublishErr,
		published:      make(chan string, 4),
		disconnected:   make(chan session.Client, 1),
	}
}

func (h *recordingHandler) AuthConnect(c *session.Client) error { return nil }

func (h *recordingHandler) AuthPublish(c *session.Client, topic *string, payload *[]byte) error {
	return h.authPublishErr
}

func (h *recordingHandler) AuthSubscribe(c *session.Client, topics *[]string) error { return nil }
func (h *recordingHandler) Connect(c *session.Client)                               {}

func (h *recordingHandler) Publish(c *session.Client, topic *string, payload *[]byte) {
	h.published <- *topic
}

func (h *recordingHandler) Subscribe(c *session.Client, topics *[]string)   {}
func (h *recordingHandler) Unsubscribe(c *session.Client, topics *[]string) {}
func (h *recordingHandler) Disconnect(c *session.Client)                    { h.disconnected <- *c }

// proxy wires a Session between a fake client and a fake broker over in-memory
// pipes, so tests can drive real MQTT packets through the public Stream API and
// observe what reaches the broker.
type proxy struct {
	client  net.Conn
	broker  net.Conn
	handler *recordingHandler
}

func newProxy(t *testing.T, h *recordingHandler) *proxy {
	t.Helper()

	clientConn, proxyIn := net.Pipe()
	proxyOut, brokerConn := net.Pipe()

	log, err := logger.New(io.Discard, "error")
	assert.Nil(t, err)

	s := session.New(proxyIn, proxyOut, h, log, x509.Certificate{})
	go s.Stream()

	t.Cleanup(func() {
		clientConn.Close()
		brokerConn.Close()
	})

	return &proxy{client: clientConn, broker: brokerConn, handler: h}
}

// send writes a packet from the client into the proxy.
func (p *proxy) send(t *testing.T, pkt packets.ControlPacket) {
	t.Helper()
	go pkt.Write(p.client)
}

// readBroker returns the next packet the broker receives, or nil if none arrives
// before the deadline.
func (p *proxy) readBroker(t *testing.T) packets.ControlPacket {
	t.Helper()

	assert.Nil(t, p.broker.SetReadDeadline(time.Now().Add(readTimeout)))
	pkt, err := packets.ReadPacket(p.broker)
	if err != nil {
		return nil
	}
	return pkt
}

func connectPacket(willTopic string) *packets.ConnectPacket {
	p := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
	p.ClientIdentifier = "client-1"
	if willTopic != "" {
		p.WillFlag = true
		p.WillTopic = willTopic
		p.WillMessage = []byte("will-payload")
	}
	return p
}

func publishPacket(topic string, dup bool) *packets.PublishPacket {
	p := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	p.TopicName = topic
	p.Payload = []byte("payload")
	p.Qos = 1
	p.Dup = dup
	return p
}

// An unauthorized Will must never reach the broker: if the CONNECT gets through,
// the broker publishes the Will, unchecked, on abnormal termination.
func TestStreamRejectsUnauthorizedWill(t *testing.T) {
	h := newRecordingHandler(errUnauthorized)
	p := newProxy(t, h)

	p.send(t, connectPacket("groups/victim-group/commands"))

	assert.Nil(t, p.readBroker(t), "unauthorized will: CONNECT must not be forwarded to the broker")

	// The Will must also not be recorded on the client. Otherwise teardown would
	// mirror the unauthorized Will onto the internal bus despite the refused connection.
	select {
	case c := <-h.disconnected:
		assert.False(t, c.WillFlag, "unauthorized will must not be recorded on the client")
	case <-time.After(time.Second):
		t.Fatal("session did not disconnect")
	}
}

// An authorized Will is forwarded to the broker intact.
func TestStreamForwardsAuthorizedWill(t *testing.T) {
	h := newRecordingHandler(nil)
	p := newProxy(t, h)

	p.send(t, connectPacket("things/thing-1/messages"))

	cp, ok := p.readBroker(t).(*packets.ConnectPacket)
	assert.True(t, ok, "authorized will: CONNECT must reach the broker")
	assert.True(t, cp.WillFlag)
	assert.Equal(t, "things/thing-1/messages", cp.WillTopic)
	assert.Equal(t, []byte("will-payload"), cp.WillMessage)
}

// A retransmission is still forwarded to the broker verbatim - the broker owns the
// QoS handshake - but the duplicate publish onto the internal bus is suppressed.
func TestStreamForwardsPublishSkippingDupOnBus(t *testing.T) {
	cases := []struct {
		desc           string
		dup            bool
		wantBusPublish bool
	}{
		{
			desc:           "first delivery reaches the broker and the bus",
			dup:            false,
			wantBusPublish: true,
		},
		{
			desc:           "retransmit reaches the broker but not the bus",
			dup:            true,
			wantBusPublish: false,
		},
	}

	for _, tc := range cases {
		h := newRecordingHandler(nil)
		p := newProxy(t, h)

		p.send(t, publishPacket("things/thing-1/messages", tc.dup))

		pp, ok := p.readBroker(t).(*packets.PublishPacket)
		assert.True(t, ok, "%s: PUBLISH must always be forwarded to the broker", tc.desc)
		assert.Equal(t, tc.dup, pp.Dup, tc.desc)

		select {
		case topic := <-h.published:
			assert.True(t, tc.wantBusPublish, "%s: unexpected bus publish of %q", tc.desc, topic)
		case <-time.After(200 * time.Millisecond):
			assert.False(t, tc.wantBusPublish, "%s: expected a bus publish, got none", tc.desc)
		}
	}
}

// An explicit DISCONNECT marks the disconnect clean, so the Will is not fired.
func TestStreamMarksCleanDisconnect(t *testing.T) {
	h := newRecordingHandler(nil)
	p := newProxy(t, h)

	p.send(t, connectPacket(""))
	assert.NotNil(t, p.readBroker(t), "CONNECT must reach the broker")

	p.send(t, packets.NewControlPacket(packets.Disconnect))
	assert.NotNil(t, p.readBroker(t), "DISCONNECT must reach the broker")

	p.client.Close()

	select {
	case c := <-h.disconnected:
		assert.True(t, c.CleanDisconnect, "explicit DISCONNECT must mark a clean disconnect")
	case <-time.After(time.Second):
		t.Fatal("session did not disconnect")
	}
}
