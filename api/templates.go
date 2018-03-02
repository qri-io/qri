package api

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

// templateRenderer returns a func "renderTemplate" that renders a template, using the values of a Config
func renderTemplate(c *Config, w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl, map[string]interface{}{
		"webappScripts": c.WebappScripts,
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
