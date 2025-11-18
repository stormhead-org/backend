package template

import (
	"bytes"
	"html/template"
)

// Render executes a template with the given data and returns the result as a string.
func Render(templatePath string, data interface{}) (string, error) {
	// Read and parse the template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}

	// Execute the template with the provided data
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}