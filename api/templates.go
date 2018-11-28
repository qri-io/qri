package api

import (
	"html/template"
	"net/http"

	"github.com/qri-io/qri/config"
)

// templates is a collection of views for rendering with the renderTemplate function
// see homeHandler for an example
var templates *template.Template

func init() {
	templates = template.Must(template.New("webapp").Parse(webapptmpl))
}

// templateRenderer returns a func "renderTemplate" that renders a template, using the values of a Config
func renderTemplate(c *config.Webapp, w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl, map[string]interface{}{
		"port": c.Port,
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
