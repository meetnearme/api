package components

import (
	"bytes"
	"html/template"
	"testing"
)

var profileLeftNavContentsTemplate = `
{{ define "ProfileLeftNavContents" }}
<li>
	<a href="/admin/profile">Profile</a>
	<ul>
		<li><a href="/admin/profile/settings">Settings</a></li>
	</ul>
</li>
<li>
	<a>Events</a>
	<ul>
		<li><a>Add an Event (Soon)</a></li>
		<li><a>Hosted Events (Soon)</a></li>
	</ul>
</li>
{{ end }}
`

var profileNavTemplate = `
{{ define "ProfileNav" }}
<div class="self-start sticky top-0 col-span-2 md:mr-5 mb-5 card border-2 border-base-300 bg-base-200 rounded-box md:place-items-center ">
	<ul class="menu bg-base-200 rounded-box w-56">
		{{ template "ProfileLeftNavContents" . }}
	</ul>
</div>
{{ end }}
`

func TestProfileLeftNavContents(t *testing.T) {
	tmpl := template.Must(template.New("profileLeftNavContents").Parse(profileLeftNavContentsTemplate))

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ProfileLeftNavContents", nil); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	result := buf.String()
	expected := `<li>
	<a href="/admin/profile">Profile</a>
	<ul>
		<li><a href="/admin/profile/settings">Settings</a></li>
	</ul>
</li>
<li>
	<a>Events</a>
	<ul>
		<li><a>Add an Event (Soon)</a></li>
		<li><a>Hosted Events (Soon)</a></li>
	</ul>
</li>`

	// Check if the result contains the expected structure
	if !bytes.Contains([]byte(result), []byte(expected)) {
		t.Errorf("expected %q to be in result, got %q", expected, result)
	}
}

func TestProfileNav(t *testing.T) {
	tmpl := template.Must(template.New("profileNav").Parse(profileNavTemplate + profileLeftNavContentsTemplate))

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ProfileNav", nil); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	result := buf.String()
	expectedDiv := `<div class="self-start sticky top-0 col-span-2 md:mr-5 mb-5 card border-2 border-base-300 bg-base-200 rounded-box md:place-items-center ">`
	expectedUL := `<ul class="menu bg-base-200 rounded-box w-56">`

	// Check if the result contains the expected div and ul structure
	if !bytes.Contains([]byte(result), []byte(expectedDiv)) {
		t.Errorf("expected %q to be in result, got %q", expectedDiv, result)
	}
	if !bytes.Contains([]byte(result), []byte(expectedUL)) {
		t.Errorf("expected %q to be in result, got %q", expectedUL, result)
	}
}

