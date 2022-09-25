# Run go golanci-lint
lint:
	golangci-lint run --fix -D typecheck

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test: tidy lint
	go test ./...  -coverprofile=coverage.out
	go tool cover -func=coverage.out

release: semver
	@version=$$(semver); \
	git tag -s $$version -m"Release $$version"
	goreleaser --rm-dist

test-release:
	goreleaser --skip-publish --snapshot --rm-dist

