// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

// CertIssued signals that a certificate was issued for a thing, without the key or certificate.
type CertIssued struct {
	Serial  string
	ThingID string
}

func (e CertIssued) Operation() string {
	return CertIssue
}

func (e CertIssued) Encode() map[string]any {
	return redisEvent{
		"serial":   e.Serial,
		"thing_id": e.ThingID,
	}
}

func decodeCertIssued(e redisEvent) CertIssued {
	return CertIssued{
		Serial:  e.field("serial", ""),
		ThingID: e.field("thing_id", ""),
	}
}

// CertRotated signals that a user replaced a thing's certificate with a newly issued one.
type CertRotated struct {
	Serial        string
	RevokedSerial string
	ThingID       string
}

func (e CertRotated) Operation() string {
	return CertRotate
}

func (e CertRotated) Encode() map[string]any {
	return redisEvent{
		"serial":         e.Serial,
		"revoked_serial": e.RevokedSerial,
		"thing_id":       e.ThingID,
	}
}

func decodeCertRotated(e redisEvent) CertRotated {
	return CertRotated{
		Serial:        e.field("serial", ""),
		RevokedSerial: e.field("revoked_serial", ""),
		ThingID:       e.field("thing_id", ""),
	}
}

// CertRevoked signals that a certificate was revoked.
type CertRevoked struct {
	Serial  string
	ThingID string
}

func (e CertRevoked) Operation() string {
	return CertRevoke
}

func (e CertRevoked) Encode() map[string]any {
	return redisEvent{
		"serial":   e.Serial,
		"thing_id": e.ThingID,
	}
}

func decodeCertRevoked(e redisEvent) CertRevoked {
	return CertRevoked{
		Serial:  e.field("serial", ""),
		ThingID: e.field("thing_id", ""),
	}
}

// CertRenewed signals that an expiring certificate was replaced, by a user or by the renewal scheduler.
type CertRenewed struct {
	Serial        string
	RevokedSerial string
	ThingID       string
}

func (e CertRenewed) Operation() string {
	return CertRenew
}

func (e CertRenewed) Encode() map[string]any {
	return redisEvent{
		"serial":         e.Serial,
		"revoked_serial": e.RevokedSerial,
		"thing_id":       e.ThingID,
	}
}

func decodeCertRenewed(e redisEvent) CertRenewed {
	return CertRenewed{
		Serial:        e.field("serial", ""),
		RevokedSerial: e.field("revoked_serial", ""),
		ThingID:       e.field("thing_id", ""),
	}
}
