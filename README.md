# Loosey (Goose)y [![Go Reference](https://pkg.go.dev/badge/github.com/typelate/loosey.svg)](https://pkg.go.dev/github.com/typelate/loosey) [![Go](https://github.com/typelate/loosey/actions/workflows/ci.yml/badge.svg)](https://github.com/typelate/loosey/actions/workflows/ci.yml)

Get [pressly/goose](https://github.com/pressly/goose) migrations without transitive dependencies.

You can continue using the `goose` CLI for manual intervention but consider using `loosey` in your app.

This is not a Go level drop in replacement. It does support your migrations files but wiring it up to your go app should be simple.

Tested against:
- PostgreSQL: 16, 17, 18
- MariaDB: 11.4, 11.7
- SQLite3: [modernc](https://pkg.go.dev/modernc.org/sqlite)
- LibSQL

_(see [integration tests](./internal/integrations) for details)_

## Development

`go generate` requires [sqlc](https://sqlc.dev) and [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) be your path.
