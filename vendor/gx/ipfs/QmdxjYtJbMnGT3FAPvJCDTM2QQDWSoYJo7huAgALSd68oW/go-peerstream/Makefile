test: deps
	go test ./muxtest

test_race: deps
	go test -race ./muxtest

gx-bins:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go

deps: gx-bins
	gx --verbose install --global
	gx-go rewrite

clean: gx-bins
	gx-go rewrite --undo
