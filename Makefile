start:
	WHATEVER_ENV=dev PORT=9998 air

build:
	go build -o ./tmp/main .

test:
	go test ./...

tag := $(shell cat ./.version)

tag:
	git diff-index --quiet HEAD -- # stop if tree is not clean
	git merge-base --is-ancestor HEAD origin/main # stop if HEAD is not pushed
	git tag ${tag}
	git push origin ${tag}
