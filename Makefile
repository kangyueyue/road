tidy:
	go mod tidy
	@ echo "go mod tidy done"

format:
	goimports -w .
	@ echo "goimports done"