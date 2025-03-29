// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smpp

import (
	"regexp"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

var _ notifiers.Notifier = (*notifier)(nil)

type notifier struct {
	transmitter   *smpp.Transmitter
	transformer   transformers.Transformer
	sourceAddrTON uint8
	sourceAddrNPI uint8
	destAddrTON   uint8
	destAddrNPI   uint8
	from          string
}

// New instantiates SMTP message notifier.
func New(cfg Config, from string) notifiers.Notifier {
	t := &smpp.Transmitter{
		Addr:        cfg.Address,
		User:        cfg.Username,
		Passwd:      cfg.Password,
		SystemType:  cfg.SystemType,
		RespTimeout: 3 * time.Second,
	}
	t.Bind()
	ret := &notifier{
		transmitter:   t,
		transformer:   json.New(),
		sourceAddrTON: cfg.SourceAddrTON,
		destAddrTON:   cfg.DestAddrTON,
		sourceAddrNPI: cfg.SourceAddrNPI,
		destAddrNPI:   cfg.DestAddrNPI,
		from:          from,
	}
	return ret
}

func (n *notifier) Notify(to []string, msg protomfx.Message) error {
	send := &smpp.ShortMessage{
		Src:           n.from,
		DstList:       to,
		Validity:      10 * time.Minute,
		SourceAddrTON: n.sourceAddrTON,
		DestAddrTON:   n.destAddrTON,
		SourceAddrNPI: n.sourceAddrNPI,
		DestAddrNPI:   n.destAddrNPI,
		Text:          pdutext.Raw(msg.Payload),
		Register:      pdufield.NoDeliveryReceipt,
	}
	_, err := n.transmitter.Submit(send)
	if err != nil {
		return err
	}
	return nil
}

func (n *notifier) ValidateContacts(contacts []string) error {
	for _, c := range contacts {
		if !isPhoneNumber(c) {
			return apiutil.ErrInvalidContact
		}
	}
	return nil
}

func isPhoneNumber(phoneNumber string) bool {
	// phoneRegexp represent regex pattern to validate E.164 phone numbers
	var phoneRegexp = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

	return phoneRegexp.MatchString(phoneNumber)
}
