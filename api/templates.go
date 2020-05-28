package api

import (
	"html/template"
	"net/http"
)

// templates is a collection of views for rendering with the renderTemplate function
// see homeHandler for an example
var templates *template.Template

// DefaultWebappPort is the default port the web app will listen on
const DefaultWebappPort = 2505

func init() {
	templates = template.Must(template.New("webapp").Parse(webapptmpl))
}

// templateRenderer returns a func "renderTemplate" that renders a template, using the values of a Config
func renderTemplate(w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl, map[string]interface{}{
		"port": DefaultWebappPort,
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
  <script type="text/javascript" src="/webapp/main.js"></script>
</body>
</html>`
