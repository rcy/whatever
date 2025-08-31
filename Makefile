start:
	APPDIR=/tmp air

test:
	go test ./...

tag := $(shell cat ./version)

tag:
	git tag ${tag}
	git push origin ${tag}
