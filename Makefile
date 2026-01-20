.PHONY: build clean run test

build:
	go build -o schema-analyzer cmd/analyzer/main.go

clean:
	rm -f schema-analyzer
	rm -rf output/

run-sqlserver:
	./schema-analyzer scan \
		--type sqlserver \
		--conn "server=localhost;user id=sa;password=YourPass123;database=U8" \
		--output ./output

run-mysql:
	./schema-analyzer scan \
		--type mysql \
		--conn "root:password@tcp(localhost:3306)/testdb" \
		--schema testdb \
		--output ./output

test:
	go test ./...

deps:
	go mod download
	go mod tidy

install:
	go install cmd/analyzer/main.go
