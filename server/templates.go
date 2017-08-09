package server

import (
	"html/template"
	"net/http"
)

// templates is a collection of views for rendering with the renderTemplate function
// see homeHandler for an example
var templates *template.Template

func init() {
	templates = template.Must(template.New("webapp").Parse(webapptmpl))
}

// renderTemplate renders a template with the values of cfg.TemplateData
func renderTemplate(w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl, map[string]interface{}{
		// "webappScripts": cfg.WebappScripts,
		"webappScripts": []string{
			"http://localhost:4000/static/bundle.js",
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const webapptmpl = `
<!DOCTYPE html>
<html>
<head>
  <title>Qri</title>
</head>
<body>
  <div id="root"></div>
  {{ range .webappScripts }}
    <script type="text/javascript" src="{{ . }}"></script>
  {{ end }}
</body>
</html>`
