<div align="center">

<img src="https://github.com/MainfluxLabs.png" width="100" alt="MainfluxLabs logo" />

# Mainflux

### Open-Source IoT Platform

[![Go Report Card](https://goreportcard.com/badge/github.com/MainfluxLabs/mainflux)](https://goreportcard.com/report/github.com/MainfluxLabs/mainflux)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue?style=flat-square)](LICENSE)

[Documentation](https://mainfluxlabs.github.io/docs) | [Contributing](CONTRIBUTING.md) | [Releases](https://github.com/MainfluxLabs/mainflux/releases)

</div>

## Introduction

Mainflux is an open-source IoT platform written in Go. It handles device connectivity over HTTP, MQTT, WebSocket, and CoAP, and provides a full pipeline from data ingestion to storage, rules evaluation, alarms, and notifications — without requiring external tooling for each step.

Data normalization is built into the platform through Profiles, which define per-group transformation and routing rules at ingestion time. Security goes beyond standard mTLS: certificate lifecycle management includes TPM and secure element support for hardware-backed key storage. The platform runs entirely on your own infrastructure with no cloud dependency.

## Features

- **Multi-Protocol Support**: HTTP, MQTT, WebSocket, and CoAP protocol adapters.
- **Device Management**: Thing and group management with flexible metadata.
- **Profiles**: Per-profile data transformation and routing configuration, including content type, field mapping, and time normalization.
- **User Management**: Organizations, groups, platform and org invites, and role-based access control (RBAC).
- **Rules Engine**: Condition and threshold-based rules with configurable alarm, email, and SMS actions.
- **Alarms**: Real-time alarm generation and lifecycle management triggered by sensor data.
- **Scheduled Actions**: Cron-based task scheduler for automated platform operations.
- **Certificates**: X.509 certificate issuance, renewal, and revocation with full mTLS support.
- **Hardware Keys**: TPM and secure element support for external key management.
- **Notifications**: Email (SMTP) and SMS (SMPP) notifier services.
- **Data Storage**: Pluggable writers and readers for PostgreSQL, TimescaleDB, and MongoDB.
- **Backup and Restore**: Full platform backup and restore capability.
- **Observability**: Prometheus metrics, Jaeger distributed tracing, and structured logging.
- **Event Sourcing**: Redis-based event streaming across services for real-time IoT event processing.
- **CLI**: Command-line interface for platform management and development workflows.
- **Container Deployment**: Docker and Docker Compose support out of the box.

## Prerequisites

The following are needed to run Mainflux:

- [Docker](https://docs.docker.com/install/) (version 20.10+)
- [Docker Compose](https://docs.docker.com/compose/install/) (version 2.0+)

Developing Mainflux will also require:

<<<<<<< HEAD
- [Go](https://golang.org/dl/) (version 1.25.7)
- [Protobuf](https://github.com/protocolbuffers/protobuf) (version 3.x)

## Install

Once the prerequisites are installed, execute the following commands from the project's root:

## Installation

Clone the repository:

```bash
git clone https://github.com/MainfluxLabs/mainflux.git
cd mainflux
```

Run with Docker Compose:

```bash
docker compose -f docker/docker-compose.yml up -d
```

Or build and run from source:

```bash
make run
```

## Usage

**Build the CLI:**

```bash
make cli
./build/mainfluxlabs-cli version
```

**Check service health:**

```bash
curl -X GET http://localhost:8080/health
```

For full API documentation and usage examples, visit [mainfluxlabs.github.io/docs](https://mainfluxlabs.github.io/docs).

## Documentation

Complete documentation is available at [mainfluxlabs.github.io/docs](https://mainfluxlabs.github.io/docs).

## Community and Contributing

Contributions are welcome and encouraged:

- [Open Issues](https://github.com/MainfluxLabs/mainflux/issues)
- [Contribution Guide](CONTRIBUTING.md)

## Authors

This project is a fork of [mainflux/mainflux](https://github.com/mainflux/mainflux), which has since been archived. It is maintained independently by [MainfluxLabs](https://github.com/MainfluxLabs), continuing the original work with new features and long-term support. See [MAINTAINERS](MAINTAINERS) for the current team.

## License

Mainflux is open-source software licensed under the [Apache-2.0](LICENSE) license. Contributions are welcome and encouraged!
