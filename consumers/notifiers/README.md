# Notifiers

Notifiers is a shared library — not a standalone service. It provides the core logic for managing notifier records and delivering notifications, and is consumed by the [SMTP Notifier](smtp/README.md) and [SMPP Notifier](smpp/README.md) services. It is composed of two layers:

1. **Notifier manager** — an HTTP API for creating, updating, listing, and removing notifier records. Notifiers are group-scoped: each notifier belongs to a group and carries a list of contact addresses (emails, phone numbers, etc.).
2. **Sender backends** — pluggable delivery implementations that consume broker messages and send notifications to the contacts defined on the relevant notifiers. Two backends are provided: [SMTP](smtp/README.md) (email) and [SMPP](smpp/README.md) (SMS).

## Data Model

| Field      | Type          | Description                                        |
|------------|---------------|----------------------------------------------------|
| `id`       | UUID          | Unique notifier identifier                         |
| `group_id` | UUID          | Group this notifier belongs to                     |
| `name`     | string        | Human-readable notifier name                       |
| `contacts` | []string      | Delivery addresses (emails, phone numbers, etc.)   |
| `metadata` | JSON object   | Arbitrary metadata                                 |

## API

The notifier manager exposes an HTTP API. Full reference: [API documentation](https://mainfluxlabs.github.io/docs/swagger/).

| Method | Path                                  | Description                              |
|--------|---------------------------------------|------------------------------------------|
| POST   | `/groups/{groupId}/notifiers`         | Create notifiers for a group             |
| GET    | `/groups/{groupId}/notifiers`         | List all notifiers for a group           |
| POST   | `/groups/{groupId}/notifiers/search`  | Search notifiers with pagination/filters |
| GET    | `/notifiers/{notifierId}`             | Get a single notifier                    |
| PUT    | `/notifiers/{notifierId}`             | Update a notifier                        |
| PATCH  | `/notifiers`                          | Remove notifiers by ID list              |

All requests require a Bearer token (`Authorization: Bearer <user_token>`).

## Sender Backends

Each backend is a separate binary that subscribes to the broker and dispatches notifications:

| Backend | Binary                          | Protocol | Contacts format       |
|---------|---------------------------------|----------|-----------------------|
| SMTP    | `mainfluxlabs-smtp-notifier`    | Email    | `user@example.com`    |
| SMPP    | `mainfluxlabs-smpp-notifier`    | SMS      | `+1234567890`         |

See the individual backend READMEs for configuration details:
- [SMTP Notifier](smtp/README.md)
- [SMPP Notifier](smpp/README.md)

## Usage

Start a sender backend to begin consuming messages from the broker and dispatching notifications to the contacts listed on the relevant notifiers. The notifier manager HTTP API can be used independently to manage notifier records.

[doc]: https://mainfluxlabs.github.io/docs
