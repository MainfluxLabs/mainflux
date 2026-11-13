# WebRTC Service

The WebRTC service is a Selective Forwarding Unit (SFU) built on [Pion](https://github.com/pion/webrtc). Things (for example cameras) publish live media to it, and users with viewer access to the thing watch that media.

Sessions are kept in memory in a single instance and are lost on restart. Nothing is recorded or persisted, and there is no database.

## How it works

- **Devices** signal over reserved `webrtc.*` subtopics on the WS and MQTT adapters. The adapters divert those messages to the broker subject `webrtc.<thingID>.<kind>`, and the WebRTC service consumes them. Replies are sent to the device as commands on `things.<thingID>.commands.<subtopic>`.
- **Viewers** signal over the HTTP endpoints of this service, authorized with a user token that must have viewer access to the thing.
- One publisher session exists per thing. A new offer replaces the previous session and closes its viewers.
- Pion's default codecs are used.

## Device signaling

The device publishes on the WS or MQTT adapter with subtopic `webrtc.<kind>`:

| Subtopic           | Payload                                                                              |
|--------------------|--------------------------------------------------------------------------------------|
| `webrtc.config`    | Any non-empty body, for example `{}`                                                 |
| `webrtc.offer`     | `{"session_id": "", "sdp": "<offer>"}`                                               |
| `webrtc.candidate` | `{"session_id": "<id>", "candidate": {"candidate": "...", "sdpMid": "0", "sdpMLineIndex": 0}}` |

The service replies with a command (`Protocol: webrtc`):

| Command subtopic   | Payload                                                  |
|--------------------|----------------------------------------------------------|
| `webrtc.config`    | `{"ice_servers": [...]}`                                 |
| `webrtc.answer`    | `{"session_id": "...", "sdp": "<answer>", "ice_servers": [...]}` |
| `webrtc.candidate` | `{"session_id": "...", "candidate": {...}}` (one per server ICE candidate, trickle ICE) |

Any other `webrtc.*` kind is rejected. Failures on this path are logged and no error reply is sent to the device.

MQTT devices only receive commands when the MQTT adapter runs with `MF_MQTT_ADAPTER_FORWARDER=true`.

## Viewer API

All endpoints require `Authorization: Bearer <user token>`. See `api/openapi/webrtc.yml`.

| Endpoint                                  | Description                                                                       |
|-------------------------------------------|-----------------------------------------------------------------------------------|
| `GET /webrtc/ice-servers`                 | Returns the ICE servers, with short-lived TURN credentials when TURN is configured |
| `POST /webrtc/things/{id}/view`           | Body `{"sdp": "<offer>"}`. Returns `{"session_id", "sdp"}` once ICE gathering is complete |
| `PATCH /webrtc/sessions/{id}/candidates`  | Body `{"candidate": {...}}`. Adds a viewer ICE candidate                          |

Viewing a thing without an active publisher returns 404. The server does not trickle candidates to viewers.

## TURN credentials

When `MF_WEBRTC_TURN_URLS` and `MF_WEBRTC_TURN_SECRET` are set, a TURN entry is added to the ICE servers using the TURN REST API scheme:

```
username   = <unix expiry>:<userID or thingID>
credential = base64(HMAC-SHA1(MF_WEBRTC_TURN_SECRET, username))
```

The expiry is now plus `MF_WEBRTC_TURN_TTL`. The compose file does not ship coturn. Configure your coturn with:

```
use-auth-secret
static-auth-secret=<same value as MF_WEBRTC_TURN_SECRET>
```

## Configuration

Any unset variable is replaced with its default value.

| Variable                      | Description                                                                         | Default                          |
|-------------------------------|-------------------------------------------------------------------------------------|----------------------------------|
| `MF_WEBRTC_LOG_LEVEL`         | Log level (debug, info, warn, error)                                                | error                            |
| `MF_WEBRTC_HTTP_PORT`         | HTTP port                                                                           | 9032                             |
| `MF_WEBRTC_UDP_PORT`          | Single UDP port all ICE traffic is multiplexed over                                 | 50000                            |
| `MF_WEBRTC_SERVER_CERT`       | Path to server certificate in PEM format                                            |                                  |
| `MF_WEBRTC_SERVER_KEY`        | Path to server key in PEM format                                                    |                                  |
| `MF_WEBRTC_CLIENT_TLS`        | Flag that indicates if TLS should be turned on for gRPC clients                     | false                            |
| `MF_WEBRTC_CA_CERTS`          | Path to trusted CAs in PEM format                                                   |                                  |
| `MF_WEBRTC_PUBLIC_IPS`        | Comma separated public IPs advertised as ICE host candidates (NAT 1:1)              |                                  |
| `MF_WEBRTC_STUN_URLS`         | Comma separated STUN URLs                                                           | stun:stun.l.google.com:19302     |
| `MF_WEBRTC_TURN_URLS`         | Comma separated TURN URLs. TURN is disabled when empty                              |                                  |
| `MF_WEBRTC_TURN_SECRET`       | Shared secret with coturn. TURN is disabled when empty                              |                                  |
| `MF_WEBRTC_TURN_TTL`          | Lifetime of TURN credentials                                                        | 24h                              |
| `MF_BROKER_URL`               | Message broker URL                                                                  | nats://localhost:4222            |
| `MF_JAEGER_URL`               | Jaeger server URL. Leave empty to disable tracing                                   |                                  |
| `MF_THINGS_AUTH_GRPC_URL`     | Things auth service gRPC URL                                                        | localhost:8183                   |
| `MF_THINGS_AUTH_GRPC_TIMEOUT` | Things auth service gRPC request timeout                                            | 1s                               |
| `MF_AUTH_GRPC_URL`            | Auth service gRPC URL                                                               | localhost:8181                   |
| `MF_AUTH_GRPC_TIMEOUT`        | Auth service gRPC request timeout                                                   | 1s                               |

## Deployment

The compose service `mainfluxlabs-webrtc` publishes the HTTP port and the UDP port (`50000/udp`). Set `MF_WEBRTC_PUBLIC_IPS` to the address clients reach the host on when the service is behind NAT.

The broker stream must include the `webrtc.>` subject. A stream created by an older version is extended with the missing subjects on startup.
