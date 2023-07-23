all: clean sqlc build 
build: 
	cd cmd/shpong;go build -o ../../bin/shpong
vendor: clean vendorbuild 
vendorbuild:
	go build -mod=vendor -o bin/shpong cmd/shpong/main.go
clean: 
	rm -f bin/shpong
modd:
	-modd
sqlc:
	-cd db/matrix;sqlc generate --experimental;
views:
	#-cd db/matrix/views;./create.sh;
	./bin/shpong views;
deps:
	-go install github.com/kyleconroy/sqlc/cmd/sqlc@latest;
	-go get -d github.com/cortesi/modd/cmd/modd;
