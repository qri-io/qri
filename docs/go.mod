module github.com/qri-io/qri/docs

go 1.15

replace github.com/qri-io/qri => ../

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/getkin/kin-openapi v0.55.0
	github.com/iancoleman/orderedmap v0.2.0
	github.com/qri-io/ioes v0.1.1
	github.com/qri-io/qfs v0.6.1-0.20210809192005-052457575e43 // indirect
	github.com/qri-io/qri v0.10.0
	github.com/qri-io/starlib v0.5.1-0.20210920143842-cd3dd3df5aa5 // indirect
	github.com/spf13/cobra v1.1.3
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	gopkg.in/yaml.v2 v2.4.0
)
