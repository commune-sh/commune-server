all: clean sqlc build 
build: 
	cd cmd/commune;go build -o ../../bin/commune
vendor: clean vendorbuild 
vendorbuild:
	go build -mod=vendor -o bin/commune cmd/commune/main.go
clean: 
	rm -f bin/commune
modd:
	-modd
sqlc:
	-cd db/matrix;sqlc generate;
views:
	#-cd db/matrix/views;./create.sh;
	./bin/commune views;
deps:
	-go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest;
	-go install github.com/cortesi/modd/cmd/modd@latest;
