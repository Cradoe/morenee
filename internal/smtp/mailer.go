package smtp

import (
	"bytes"
	"time"

	"github.com/cradoe/morenee/assets"
	"github.com/cradoe/morenee/internal/funcs"

	"github.com/wneessen/go-mail"

	htmlTemplate "html/template"
	textTemplate "text/template"
)

const defaultTimeout = 10 * time.Second

type MailClient interface {
	DialAndSend(...*mail.Msg) error
}

type MailerInterface interface {
	Send(recipient string, data any, patterns ...string) error
}
type Mailer struct {
	client MailClient
	from   string
}

func NewMailer(host string, port int, username, password, from string) (*Mailer, error) {
	client, err := mail.NewClient(
		host,
		mail.WithTimeout(defaultTimeout),
		mail.WithPort(port),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		return nil, err
	}

	return &Mailer{
		client: client, // Now using the interface
		from:   from,
	}, nil
}

func (m *Mailer) Send(recipient string, data any, patterns ...string) error {
	for i := range patterns {
		patterns[i] = "emails/" + patterns[i]
	}
	msg := mail.NewMsg()

	if err := msg.To(recipient); err != nil {
		return err
	}

	if err := msg.From(m.from); err != nil {
		return err
	}

	ts, err := textTemplate.New("").Funcs(funcs.TemplateFuncs).ParseFS(assets.EmbeddedFiles, patterns...)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	if err := ts.ExecuteTemplate(subject, "subject", data); err != nil {
		return err
	}

	msg.Subject(subject.String())

	plainBody := new(bytes.Buffer)
	if err := ts.ExecuteTemplate(plainBody, "plainBody", data); err != nil {
		return err
	}

	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())

	if ts.Lookup("htmlBody") != nil {
		ts, err := htmlTemplate.New("").Funcs(funcs.TemplateFuncs).ParseFS(assets.EmbeddedFiles, patterns...)
		if err != nil {
			return err
		}

		htmlBody := new(bytes.Buffer)
		if err := ts.ExecuteTemplate(htmlBody, "htmlBody", data); err != nil {
			return err
		}

		msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())
	}

	for i := 1; i <= 3; i++ {
		err := m.client.DialAndSend(msg) // Now calls interface method

		if err == nil {
			return nil
		}

		if i != 3 {
			time.Sleep(2 * time.Second)
		}
	}

	return err
}
