# go-countries

`go-countries` is a Go package for the ISO 3166-1 country list and the
groupings that decide where data is allowed to live.

It answers the questions that come up when a region string has been resolved
to a country code and something has to be done about it: is this a real
country, what continent is it on, and is it inside the EU or the EEA.

## Features

* The full ISO 3166-1 list, 249 entries, embedded in the binary
* Lookup by alpha-2, alpha-3 or numeric code, in any case (`FR`, `fra`, `250`)
* UN M49 region and sub-region for each country
* EU (27) and EEA (30) membership
* Formal ISO names alongside the common ones (`United Kingdom`, not
  `United Kingdom of Great Britain and Northern Ireland`)
* No dependencies, no data files, no network access

## Install

```sh
go get github.com/PlakarKorp/go-countries
```

## Usage

```go
package main

import (
	"fmt"

	countries "github.com/PlakarKorp/go-countries"
)

func main() {
	c, ok := countries.Get("fr")
	if !ok {
		return
	}

	fmt.Println(c.CommonName()) // France
	fmt.Println(c.Alpha3)       // FRA
	fmt.Println(c.Numeric)      // 250
	fmt.Println(c.Region)       // Europe
	fmt.Println(c.SubRegion)    // Western Europe
	fmt.Println(c.EU, c.EEA)    // true true

	fmt.Println(countries.IsEU("GB"))  // false
	fmt.Println(countries.IsEEA("CH")) // false, EFTA but not EEA

	fmt.Println(len(countries.All()))                          // 249
	fmt.Println(len(countries.EUMembers()))                    // 27
	fmt.Println(len(countries.InRegion(countries.RegionEurope)))
}
```

Validating a code without caring which country it is:

```go
if !countries.Valid(code) {
	return fmt.Errorf("%q is not an ISO 3166-1 country", code)
}
```

## Notes

`Get` reports `ok=false` for a code it does not know rather than substituting a
default, because what to do about an unknown country is the caller's decision.

The UN M49 classification does not cover every ISO country: Antarctica (`AQ`)
and Taiwan (`TW`) have no region, and their `Region` and `SubRegion` are empty.
That is unknown, not a bug.

`Name` is the ISO short name, which for some countries is the formal one.
`CommonName()` gives the everyday spelling.

`EU` and `EEA` are political facts rather than ISO ones, so they are maintained
in the generator rather than read from the source data. The tests assert their
cardinality, so a bad edit fails the build.

## Regenerating the data

`countries_data.go` is generated. See [internal/gen](internal/gen/README.md).

## License

ISC, see [LICENSE](LICENSE).
