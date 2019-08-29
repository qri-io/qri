default: test

test:
	go test ./... -v --coverprofile=coverage.txt --covermode=atomic
	
update-changelog:
	conventional-changelog -p angular -i CHANGELOG.md -s