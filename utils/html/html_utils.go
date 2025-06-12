package utils

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
	"time"
)

// GenerateHTMLContent generates HTML content by embedding the parsed content into the given template
func GenerateHTMLContent(subject, content string) (string, error) {
	// Define the HTML template
	tmpl := `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f4f4f4;
            margin: 0;
            padding: 0;
        }

        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            padding: 20px;
            border-radius: 5px;
        }

        .header {
            background-color: #007bff;
            color: white;
            padding: 10px 0;
            text-align: center;
            border-radius: 5px 5px 0 0;
            font-size: 1.2em;
        }

        .content {
            padding: 20px;
            color: #333333;
            font-size: 14px;
            line-height: 1.6;
            margin-bottom: 20px;
            background-color: #f9f9f9;
            border-radius: 5px;
        }

        .footer {
            text-align: center;
            padding: 10px;
            font-size: 12px;
            color: #777777;
        }

        a {
            color: #007bff;
            text-decoration: none;
        }

        a:hover {
            text-decoration: underline;
        }
    </style>
</head>

<body>
    <div class="container">
        <div class="header">
            <h1>{{ .Subject }}</h1>
        </div>
        <div class="content">
            {{ .Content }}
        </div>
        <div class="footer">
            <p>&copy; {{ .Year }} idealAi. All rights reserved.</p>
        </div>
    </div>
</body>

</html>
`

	// Parse the template
	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", err
	}

	// Replace Markdown-style formatting with HTML tags
	htmlContent := strings.ReplaceAll(content, "\n\n", "<br><br>")
	htmlContent = strings.ReplaceAll(htmlContent, "**", "") // Remove bold formatting
	htmlContent = strings.ReplaceAll(htmlContent, "###", "<p>")
	htmlContent = strings.ReplaceAll(htmlContent, "---", "<hr>")

	// Replace bullet points correctly
	htmlContent = strings.ReplaceAll(htmlContent, "- ", "<li>")
	htmlContent = strings.ReplaceAll(htmlContent, "</li><br><br>", "</li>") // Fix list closure

	// Detect URLs and convert them into clickable links
	urlRegex := regexp.MustCompile(`(https?://[^\s]+)`)
	htmlContent = urlRegex.ReplaceAllString(htmlContent, `<a href="$1" target="_blank">$1</a>`)

	// Data to pass into the template
	data := struct {
		Subject string
		Content template.HTML
		Year    int
	}{
		Subject: subject,
		Content: template.HTML(htmlContent), // Treat content as safe HTML
		Year:    time.Now().Year(),
	}

	// Generate the HTML output
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
