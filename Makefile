test:
	go vet ./...
	go test -trimpath -race ./...

update-spec:
	curl --fail --silent --show-error --location --output open_api_spec.yaml https://api.ynab.com/papi/open_api_spec.yaml

release:
	bump_version --tag-prefix=v minor client.go
