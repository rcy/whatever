start:
	APPDIR=/tmp air

test:
	go test ./...

tag := $(shell cat ./VERSION)

tag:
	git diff-index --quiet HEAD -- # error if tree is not clean
	git tag ${tag}
	git push origin ${tag}
