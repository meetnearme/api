// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.543
package pages

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

func SignUpPage() templ.Component {
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
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<script async crossorigin=\"anonymous\" data-clerk-publishable-key=\"pk_test_YnJhdmUtYmVldGxlLTEyLmNsZXJrLmFjY291bnRzLmRldiQ\" src=\"https://brave-beetle-12.clerk.accounts.dev/npm/@clerk/clerk-js@latest/dist/clerk.browser.js\" type=\"text/javascript\"></script><script>\n\t\twindow.addEventListener(\"load\", async function () {\n\t\t\tawait Clerk.load();\n\n\t\t\tconst signUpDiv =\n\t\t\tdocument.getElementById('signup-component');\n\n\t\t\tClerk.mountSignUp(signUpDiv);\n\t\t});\n\t</script><section class=\"w-full flex justify-center content-center mt-3\"><div class=\"mx-auto\" id=\"signup-component\"></div></section>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}
