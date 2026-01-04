start:
	WHATEVER_ENV=dev PORT=9998 go tool air

build:
	go build -o ./tmp/main .

test:
	go test ./...

include .env
sql:
	sqlite3 ${EVOKE_FILE}

release:
	$(if $(TAG),,$(error TAG is not defined))
	git diff-index --quiet HEAD -- # stop if tree is not clean
	git merge-base --is-ancestor HEAD origin/main # stop if HEAD is not pushed
	git tag ${TAG}
	git push origin ${TAG}

tags:
	git tag --list | tail
