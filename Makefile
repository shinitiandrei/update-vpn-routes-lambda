BINARY_NAME=updatevpnroutes

.PHONY: build
build:
	GOOS=darwin GOARCH=arm64 go build -o ${BINARY_NAME}-darwin main.go

.PHONY: run
run:
	chmod +x ./${BINARY_NAME}-darwin
	./${BINARY_NAME}-darwin

.PHONY: build_and_run
build_and_run: build run

.PHONY: clean
clean:
	go clean
	rm ${BINARY_NAME}-darwin

.PHONY: test
test:
	go test .

.PHONY: test_coverage
test_coverage:
	go test . -coverprofile=coverage.out


#### run this prior: go mod init  github.com/{username}/{project name}
.PHONY: install
install:
	go mod download
	go mod tidy

.PHONY: vet
vet:
	go vet