// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.663
package pages

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import (
	"fmt"
	"os"
	"regexp"
)

func AdminPage() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
		if !templ_7745c5c3_IsBuffer {
			templ_7745c5c3_Buffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<h2 class=\"text-3xl\">Add an Event Source</h2><br><br><div class=\"flex w-full h-full items-start\"><div class=\"w-1/2 flex-grow card border-2 border-base-300 bg-base-200 rounded-box place-items-center\"><ul id=\"event-source-steps\" class=\"steps steps-vertical\"><li class=\"step step-primary\">Add a Target URL</li><li class=\"step\">Verify Events</li><li class=\"step\">Add to Site</li></ul></div><div class=\"divider divider-horizontal\"></div><div id=\"event-source-container\" class=\"flex-grow card border-2 border-base-300 bg-base-200 p-10 rounded-box place-items-center\"><form class=\"group\" novalidate hx-post=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var2 string
		templ_7745c5c3_Var2, templ_7745c5c3_Err = templ.JoinStringErrs(templ.EscapeString(os.Getenv("SESHU_FN_URL")))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `functions/gateway/templates/pages/admin.templ`, Line: 23, Col: 89}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var2))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" hx-ext=\"json-enc\" hx-target=\"#event-candidates-inner\" hx-disabled-elt=\"input[name=&#39;url&#39;], button[type=&#39;submit&#39;]\" hx-on:submit=\"\" onSubmit=\"var $explainerNode = document.getElementById(&#39;explainer-section&#39;).classList; $explainerNode.remove(&#39;opacity-0&#39;);$explainerNode.remove(&#39;h-0&#39;);$explainerNode.add(&#39;opacity-100&#39;);$explainerNode.add(&#39;h-auto&#39;);var $eventCandidatesNode = document.getElementById(&#39;event-candidates&#39;).classList; $eventCandidatesNode.remove(&#39;h-0&#39;); $eventCandidatesNode.remove(&#39;opacity-0&#39;); $eventCandidatesNode.add(&#39;h-auto&#39;)\"><label for=\"url\">Enter a URL that lists events and we will check the site and propose some events that might be on that page (this will take some time)</label> <input name=\"url\" id=\"url\" type=\"url\" required pattern=\"")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var3 string
		templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(fmt.Sprint(regexp.MustCompile(
			`https?:\/\/(www\.)?[\w\-._~:/?#[\]@!$&'()*+,;=]+(\.[a-z]{2,})+([\w\-._~:/?#[\]@!$&'()*+,;=]*)?`,
		)))
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `functions/gateway/templates/pages/admin.templ`, Line: 32, Col: 7}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" class=\"peer border border-gray-300 p-2 w-full\" placeholder=\"Event source URL\"> <span class=\"mt-2 hidden text-sm text-red-500 peer-[&amp;:not(:placeholder-shown):not(:focus):invalid]:block\">Please enter a valid URL (must start with http:// or https://)</span><br><br><button type=\"submit\" class=\"btn btn-primary group-invalid:pointer-events-none group-invalid:opacity-30\">Search for Events</button></form><br><br><div class=\"divider divider-horizontal\"></div><div id=\"explainer-section\" aria-live=\"polite\" class=\"opacity-0 h-0 w-full transition-all\"><div role=\"alert\" class=\"alert alert-info\"><span class=\"loading-visible loading loading-ball loading-lg h-auto\"></span> <svg xmlns=\"http://www.w3.org/2000/svg\" class=\"loaded-visible stroke-current shrink-0 h-10 w-10\" fill=\"none\" viewBox=\"0 0 24 24\"><path stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z\"></path></svg><div><p class=\"loading-visible\">We are checking that URL to see if there are any events. This is going to take some time. When the process finishes, we're going to present you a list of events we think might be on that page. </p><p class=\"loaded-visible\">Here are the events we've found. </p><br><p class=\"font-bold\">What we need from you is to confirm if our guesses are accurate.</p></div></div><br><br></div><div id=\"event-candidates\" aria-live=\"polite\" class=\"w-full opacity-0 h-0 transition-all\"><h2 class=\"text-2xl font-bold\">Are these events?</h2><div id=\"event-candidates-inner\"><div class=\"grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 justify-stretch\"><div class=\"skeleton card card-compact h-96 w-full shadow-lg\"></div><div class=\"skeleton card card-compact h-96 w-full shadow-lg\"></div><div class=\"skeleton card card-compact h-96 w-full shadow-lg\"></div></div></div></div></div></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}