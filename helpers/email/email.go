package email

import (
	"bytes"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

func SendCustomEmail(receivers []string, subject string, content string, templatePath *string) error {
	// Set up authentication information
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("EMAIL_SENDER")
	smtpPassword := os.Getenv("EMAIL_SENDER_APP_PASSWORD")
	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)

	// Parse the HTML template
	var tmpl *template.Template
	var err error

	// Check if templatePath is provided; if not, use the default template path
	if templatePath == nil {
		tmpl, err = template.ParseFiles("./assets/default_email_template.html")

	} else {
		tmpl, err = template.ParseFiles(*templatePath)
	}

	if err != nil {
		return err
	}

	// Prepare the data for the template
	emailData := struct {
		Subject string
		Content string
		Year    string
	}{
		Subject: subject,
		Content: content,
		Year:    strconv.Itoa(time.Now().Year()),
	}

	// Generate the HTML content with data
	var emailBody bytes.Buffer
	if err = tmpl.Execute(&emailBody, emailData); err != nil {
		return err
	}

	// Send the email to each receiver
	for _, email := range receivers {
		go func(receiver string) {
			// Prepare email headers and body
			message := []byte("To: " + receiver + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"MIME-version: 1.0;\r\n" +
				"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
				"\r\n" + emailBody.String())

			// Send the email
			err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpUsername, []string{receiver}, message)
			if err != nil {
				log.Printf("Failed to send email to %s: %v", receiver, err)
			}
		}(email)
	}

	return nil
}
