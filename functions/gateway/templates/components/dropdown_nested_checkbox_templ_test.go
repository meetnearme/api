package components

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
)

// Mock data for testing
var mockCategories = []helpers.Category{
	{
		Name: "Category 1",
		Items: []helpers.Subcategory{
			{Name: "Subcategory 1.1"},
			{Name: "Subcategory 1.2"},
		},
	},
	{
		Name: "Category 2",
		Items: []helpers.Subcategory{
			{Name: "Subcategory 2.1"},
		},
	},
}

func init() {
	// Replace the actual helpers.Categories with mock data for tests
	helpers.Categories = mockCategories
}

// Define your templates here
var dropdownTemplate = `
{{ define "DropdownNestedCheckbox" }}
<details class="dropdown">
    {{ template "NestedCheckboxList" . }}
</details>
{{ end }}
`

var nestedCheckboxListTemplate = `
{{ define "NestedCheckboxList" }}
<summary class="m-1 btn">Categories</summary>
{{ range . }}
	<div>
		<label>{{ .Name }}</label>
		{{ range .Items }}
			<label>{{ .Name }}</label>
		{{ end }}
	</div>
{{ end }}
{{ end }}
`
func TestDropdownNestedCheckboxFromNavbar(t *testing.T) {
    // Parse both templates together
    tmpl := template.Must(template.New("dropdown").Parse(dropdownTemplate + nestedCheckboxListTemplate))

    var buf bytes.Buffer
    // Execute the DropdownNestedCheckbox template
    if err := tmpl.ExecuteTemplate(&buf, "DropdownNestedCheckbox", nil); err != nil {
        t.Fatalf("failed to execute template: %v", err)
    }

    result := buf.String()
    expected := "<details class=\"dropdown\">"
    if !bytes.Contains([]byte(result), []byte(expected)) {
        t.Errorf("expected %q to be in result, got %q", expected, result)
    }
}


func TestNestedCheckboxList(t *testing.T) {
	tmpl := template.Must(template.New("nestedCheckboxList").Parse(nestedCheckboxListTemplate))

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "NestedCheckboxList", mockCategories); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	result := buf.String()
	expectedSummary := "<summary class=\"m-1 btn\">Categories</summary>"
	// Check if the summary for the nested checkbox list is rendered
	if !bytes.Contains([]byte(result), []byte(expectedSummary)) {
		t.Errorf("expected %q to be in result, got %q", expectedSummary, result)
	}

	expectedCategory1 := "Category 1"
	expectedSubcategory1_1 := "Subcategory 1.1"
	// Check if the categories and subcategories are rendered correctly
	if !bytes.Contains([]byte(result), []byte(expectedCategory1)) ||
		!bytes.Contains([]byte(result), []byte(expectedSubcategory1_1)) {
		t.Errorf("expected categories and subcategories to be in result, got %q", result)
	}
}

