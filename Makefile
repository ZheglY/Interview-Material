.PHONY: fmt fmt-check vet test test-race quality

# fmt форматирует все рукописные Go-файлы стандартным инструментом
fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

# vet запускает статический анализатор Go
vet:
	go vet ./...

# test запускает переносимые тесты и рассчитывает покрытие
test:
	go test -cover ./...

test-race:
	go test -race -cover ./...

# quality последовательно выполняет все проверки, обязательные перед commit
quality:
	fmt-check vet test