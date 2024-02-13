# Fetch Rewards API

The Fetch Rewards API is a single Go executable that runs an HTTP server and
provides the API endpoints as defined in [api.yml](./api.yml).

## Environment

[Go 1.22](https://golang.org/dl) is **required** as we make use of the new
[enhanced support for wildcard routing
patterns](https://tip.golang.org/doc/go1.22#enhanced_routing_patterns) in the
standard [http.ServeMux](https://pkg.go.dev/net/http#ServeMux).

If you have not, or do not wish to, update to the latest Go toolchain a
Dockerfile with the required tooling has also been provided and can be used as
shown below.

```
docker build -t fetch-api-server .

# Runs the Fetch API server by default.
docker run -it --rm -p 8080:8080 fetch-api-server

# Run the tests in the Docker container.
# docker run --rm fetch-api-server go test -v github.com/admtnnr/fetch/...
```

## Building / Running

The Fetch Rewards API uses Go modules, but does not require any libraries
outside of the Go standard library. Building and running the Fetch Rewards API
can be done using any of the normal `go` commands.

```
go run github.com/admtnnr/fetch/cmd/fetch-api-server
# go build -o fetch-api-server github.com/admtnnr/fetch/cmd/fetch-api-server && ./fetch-api-server
# go install github.com/admtnnr/fetch/cmd/fetch-api-server
```

## Testing

The Fetch Rewards API comes with a suite of integration tests that leverage the
example receipts provided in the [README](./README.md) and
[examples](./examples) directory.

To run the tests, use the normal `go test` command.

```
go test github.com/admtnnr/fetch/...
# go test -v -race github.com/admtnnr/fetch/...
# go test -v -run "TestIntegration/example_simple_receipt" github.com/admtnnr/fetch/...
```

You may encounter a warning that looks like `warning: ignoring symlink
path/to/examples` when running tests. This happens because `examples` was
renamed to `testdata` to [use as fixtures for `go
test`](https://dave.cheney.net/2016/05/10/test-fixtures-in-go) and symlinked as
`examples` for backwards compatibility. When certain `go` commands such as `go
test` are run with a `/...` package pattern this warning will be printed. The
warning can be safely ignored.
