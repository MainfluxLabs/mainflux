# Notifiers

Notifiers is a shared library — not a standalone service. It provides the core logic for managing notifier records and delivering notifications, and is consumed by the [SMTP Notifier](smtp/README.md) and [SMPP Notifier](smpp/README.md) services. It is composed of two layers:

1. **Notifier manager** — an HTTP API for creating, updating, listing, and removing notifier records. Notifiers are group-scoped: each notifier belongs to a group and carries a list of contact addresses (emails, phone numbers, etc.).
2. **Sender backends** — pluggable delivery implementations that consume broker messages and send notifications to the contacts defined on the relevant notifiers. Two backends are provided: [SMTP](smtp/README.md) (email) and [SMPP](smpp/README.md) (SMS).

## Data Model

| Field      | Type        | Description                                      |
|------------|-------------|--------------------------------------------------|
| `id`       | UUID        | Unique notifier identifier                       |
| `group_id` | UUID        | Group this notifier belongs to                   |
| `name`     | string      | Human-readable notifier name                     |
| `contacts` | []string    | Delivery addresses (emails, phone numbers, etc.) |
| `metadata` | JSON object | Arbitrary metadata                               |

## Sender Backends

Each backend is a separate binary that subscribes to the broker and dispatches notifications:

| Backend | Binary                       | Protocol | Contacts format    |
|---------|------------------------------|----------|--------------------|
| SMTP    | `mainfluxlabs-smtp-notifier` | Email    | `user@example.com` |
| SMPP    | `mainfluxlabs-smpp-notifier` | SMS      | `+1234567890`      |

See the individual backend READMEs for configuration details:
- [SMTP Notifier](smtp/README.md)
- [SMPP Notifier](smpp/README.md)

## Deployment

This package is a shared library and does not have its own binary. Deploy the individual sender backends:

- [SMTP Notifier](smtp/README.md) — see deployment instructions
- [SMPP Notifier](smpp/README.md) — see deployment instructions

## Usage

Start a sender backend to begin consuming messages from the broker and dispatching notifications to the contacts listed on the relevant notifiers. The notifier manager HTTP API can be used independently to manage notifier records.

