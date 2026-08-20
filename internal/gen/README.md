# Regenerating the country table

`countries_data.go` is generated, not edited. It is derived from the ISO
3166-1 list published with UN M49 region codes:

    https://github.com/lukes/ISO-3166-Countries-with-Regional-Codes

To regenerate after an ISO revision, from the repository root:

    go generate ./...
    go test ./...

That fetches the current upstream CSV. To regenerate from a local copy
instead, which is what you want when reviewing a change to the data:

    go run ./internal/gen -source all.csv -o countries_data.go

The output is formatted with `go/format`, so there is no gofmt step, and the
generator fails rather than writing a table it cannot vouch for: an unknown
region, a duplicate country, a missing column, or a membership entry naming a
country the source does not list.

EU and EEA membership live in `main.go`, because they are political facts
rather than ISO ones and the CSV does not carry them. The generator checks
their cardinality and the package tests check it again (27 and 30), so a bad
edit fails rather than shipping.
