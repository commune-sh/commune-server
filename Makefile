all: clean sqlc build 
build: 
	cd cmd/shpong;go build -o ../../bin/shpong
setup:
	./db/matrix/views/build.sh
vendor: clean vendorbuild 
vendorbuild:
	go build -mod=vendor -o bin/shpong cmd/shpong/main.go
clean: 
	rm -f bin/shpong
rebuild: stopUnit clean build startUnit
production: stopUnit reset clean build startUnit
cleanBuild: clean build
dev: devReset clean build modd
modd:
	-modd
refresh: reset clean build run
run:
	./bin/shpong
reset: stopSynapse dropDB flushRedis createDB runMigrations startSynapse
devReset: stopDevSynapse dropDB flushRedis createDB runMigrations startDevSynapse
createDB:
	-createdb shpong
	-createdb --encoding=UTF8 --locale=C --template=template0 synapse
dropDB:
	-dropdb shpong
	-dropdb synapse
stopSynapse: SHELL := /bin/bash
stopSynapse:
	-cd ..;cd synapse;source env/bin/activate;synctl stop;
startSynapse: SHELL := /bin/bash
startSynapse:
	-cd ..;cd synapse;source env/bin/activate;synctl start;
runMigrations:
	-cd db/migrations;goose postgres "postgres://shpong:@localhost:5432/shpong?sslmode=disable" up;
resetMigrations:
	-cd db/migrations;goose postgres "postgres://shpong:@localhost:5432/shpong?sslmode=disable" reset;
sqlc:
	-cd db;sqlc generate --experimental;
	-cd db/matrix;sqlc generate --experimental;
flushRedis: SHELL := /bin/bash
flushRedis:
	-REDISCLI_AUTH=$$REDISAUTH redis-cli -n 1 flushdb
startUnit:
	-systemctl --user start shpong-dev.service
stopUnit:
	-systemctl --user stop shpong-dev.service
deps:
	-go install github.com/kyleconroy/sqlc/cmd/sqlc@latest;
	-go get -d github.com/cortesi/modd/cmd/modd;
	-go install github.com/pressly/goose/v3/cmd/goose@latest;
