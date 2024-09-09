// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package email

import (
	"bytes"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"golang.org/x/net/idna"
	"gopkg.in/gomail.v2"
)

const (
	maxLocalLen  = 64
	maxDomainLen = 255
	maxTLDLen    = 24 // longest TLD currently in existence

	atSeparator  = "@"
	dotSeparator = "."
)

var (
	// ErrMissingEmailTemplate missing email template file
	errMissingEmailTemplate = errors.New("Missing e-mail template file")
	errParseTemplate        = errors.New("Parse e-mail template failed")
	errExecTemplate         = errors.New("Execute e-mail template failed")
	errSendMail             = errors.New("Sending e-mail failed")

	userRegexp    = regexp.MustCompile("^[a-zA-Z0-9!#$%&'*+/=?^_`{|}~.-]+$")
	hostRegexp    = regexp.MustCompile(`^[^\s]+\.[^\s]+$`)
	userDotRegexp = regexp.MustCompile("(^[.]{1})|([.]{1}$)|([.]{2,})")
)

type email struct {
	To      []string
	From    string
	Subject string
	Header  string
	Content string
	Footer  string
}

// Config email agent configuration.
type Config struct {
	Host        string
	Port        string
	Username    string
	Password    string
	FromAddress string
	FromName    string
	Template    string
}

// Agent for mailing
type Agent struct {
	conf *Config
	tmpl *template.Template
	dial *gomail.Dialer
}

// New creates new email agent
func New(c *Config) (*Agent, error) {
	a := &Agent{}
	a.conf = c
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return a, err
	}
	d := gomail.NewDialer(c.Host, port, c.Username, c.Password)
	a.dial = d

	tmpl, err := template.ParseFiles(c.Template)
	if err != nil {
		return a, errors.Wrap(errParseTemplate, err)
	}
	a.tmpl = tmpl
	return a, nil
}

// Send sends e-mail
func (a *Agent) Send(To []string, From, Subject, Header, Content, Footer string) error {
	if a.tmpl == nil {
		return errMissingEmailTemplate
	}

	buff := new(bytes.Buffer)
	e := email{
		To:      To,
		From:    From,
		Subject: Subject,
		Header:  Header,
		Content: Content,
		Footer:  Footer,
	}
	if From == "" {
		from := mail.Address{Name: a.conf.FromName, Address: a.conf.FromAddress}
		e.From = from.String()
	}

	if err := a.tmpl.Execute(buff, e); err != nil {
		return errors.Wrap(errExecTemplate, err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", e.From)
	m.SetHeader("To", To...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/plain", buff.String())

	if err := a.dial.DialAndSend(m); err != nil {
		return errors.Wrap(errSendMail, err)
	}

	return nil
}

func IsEmail(email string) bool {
	if email == "" {
		return false
	}

	es := strings.Split(email, atSeparator)
	if len(es) != 2 {
		return false
	}
	local, host := es[0], es[1]

	if local == "" || len(local) > maxLocalLen {
		return false
	}

	hs := strings.Split(host, dotSeparator)
	if len(hs) < 2 {
		return false
	}
	domain, ext := hs[0], hs[1]

	// Check subdomain and validate
	if len(hs) > 2 {
		if domain == "" {
			return false
		}

		for i := 1; i < len(hs)-1; i++ {
			sub := hs[i]
			if sub == "" {
				return false
			}
			domain = fmt.Sprintf("%s.%s", domain, sub)
		}

		ext = hs[len(hs)-1]
	}

	if domain == "" || len(domain) > maxDomainLen {
		return false
	}
	if ext == "" || len(ext) > maxTLDLen {
		return false
	}

	punyLocal, err := idna.ToASCII(local)
	if err != nil {
		return false
	}
	punyHost, err := idna.ToASCII(host)
	if err != nil {
		return false
	}

	if userDotRegexp.MatchString(punyLocal) || !userRegexp.MatchString(punyLocal) || !hostRegexp.MatchString(punyHost) {
		return false
	}

	return true
}
