package components

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
)

// Mock user data for testing
var mockUserWithEmail = helpers.UserInfo{
	Name:  "John Doe",
	Email: "john@example.com",
}

var mockUserWithoutEmail = helpers.UserInfo{
	Name:  "Jane Doe",
	Email: "",
}

// Define the templates needed for the tests
func setupTemplates() *template.Template {
	tmpl := template.New("testTemplates")

	tmpl.Funcs(template.FuncMap{
		"EscapeString": func(s string) string { return s }, // Mock for EscapeString function
	})

	// Define MiniProfileInNav template
	tmpl.New("MiniProfileInNav").Parse(`
		<strong>{{ .Name }}</strong>
		<br/>
		{{ .Email }}
		<ul tabindex="0" class="menu menu-sm">
			<li>
				<a href="/admin/profile" class="justify-between">Profile</a>
			</li>
			<li><a href="/auth/logout">Logout</a></li>
		</ul>
	`)

	// Define NavListItems template
	tmpl.New("NavListItems").Parse(`
		<li><a href="/about" class="px-5 py-3">About</a></li>
		{{ if .Email }}{{ else }}
			<li><a href="/auth/login" class="btn btn-primary">Sign Up</a></li>
		{{ end }}
	`)

	// Define NestedCheckboxList template
	tmpl.New("NestedCheckboxList").Parse(`
		{{ define "NestedCheckboxList" }}
		<!-- Your checkbox list rendering logic -->
		{{ end }}
	`)

	// Define Navbar template
	tmpl.New("Navbar").Parse(`
		<div>
			{{ template "NavListItems" .UserInfo }}
		</div>
	`)

	return tmpl
}


func TestNavListItems(t *testing.T) {
	tmpl := setupTemplates()

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "NavListItems", mockUserWithoutEmail); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	result := buf.String()
	if !bytes.Contains([]byte(result), []byte("Sign Up")) {
		t.Errorf("expected 'Sign Up' link to be in result, got %q", result)
	}

	buf.Reset()
	if err := tmpl.ExecuteTemplate(&buf, "NavListItems", mockUserWithEmail); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	result = buf.String()
	if bytes.Contains([]byte(result), []byte("Sign Up")) {
		t.Errorf("did not expect 'Sign Up' link for logged-in user, got %q", result)
	}
}

