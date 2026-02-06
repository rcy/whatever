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

pull-prod:
	curl https://notnow.fly.dev
	rm -rf ./data
	mkdir ./data
	fly sftp get /data/notnow_evoke.db ./data/notnow_evoke.db
	fly sftp get /data/notnow_evoke.db-shm ./data/notnow_evoke.db-shm
	fly sftp get /data/notnow_evoke.db-wal ./data/notnow_evoke.db-wal
