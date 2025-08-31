start:
	WHATEVER_ENV=dev air

test:
	go test ./...

tag := $(shell cat ./.version)

tag:
	git diff-index --quiet HEAD -- # stop if tree is not clean
	git merge-base --is-ancestor HEAD origin/main # stop if HEAD is not pushed
	git tag ${tag}
	git push origin ${tag}
