all:
	@echo 'Available targets:'
	@echo '    make test'

test: static-tests cover.html

static-tests: FORCE
	@echo '>> gofmt...'
	@if s="$$(gofmt -s -l $$(find -name \*.go) 2>/dev/null)" && test -n "$$s"; then echo "$$s"; false; fi
	@echo '>> golint...'
	@if s="$$(golint ./... 2>/dev/null)" && test -n "$$s"; then echo "$$s"; false; fi
	@echo '>> govet...'
	@go vet ./...

cover.out: FORCE
	@echo '>> go test...'
	@go test -coverpkg github.com/majewsky/schwift/... -coverprofile $@
cover.html: cover.out
	@echo '>> rendering cover.html...'
	@go tool cover -html=$< -o $@

.PHONY: FORCE
