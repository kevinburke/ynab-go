test:
	go vet ./...
	go test -trimpath -race ./...

update-spec:
	curl --silent --fail --output open_api_spec.yaml https://api.ynab.com/papi/open_api_spec.yaml

release:
	bump_version --tag-prefix=v minor client.go
