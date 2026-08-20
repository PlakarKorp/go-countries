# Regenerating the country table

`countries_data.go` is generated, not edited. It is derived from the ISO
3166-1 list published with UN M49 region codes:

    https://github.com/lukes/ISO-3166-Countries-with-Regional-Codes

To regenerate after an ISO revision:

    curl -sLo all.csv https://raw.githubusercontent.com/lukes/ISO-3166-Countries-with-Regional-Codes/master/all/all.csv
    python3 gen.py all.csv ../../countries_data.go
    cd ../.. && gofmt -w countries_data.go && go test ./...

The EU and EEA membership sets live in `gen.py`, because they are political
facts rather than ISO ones and the CSV does not carry them. `go test` checks
their cardinality (27 and 30), so a bad edit there fails rather than shipping.
