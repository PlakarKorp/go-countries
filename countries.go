// Package countries provides the ISO 3166-1 country list together with the
// geographic and regulatory groupings that decide where data is allowed to
// live.
//
// The table is embedded in the binary: there is no data file to ship, no
// network call, and no initialisation to sequence. Lookups are map reads
// against tables built once at init.
//
// Codes are the currency of the API. A Country is identified by its alpha-2
// code ("FR"), and every lookup accepts alpha-2, alpha-3 or numeric spelling
// so callers do not have to normalise before asking.
package countries

//go:generate go run ./internal/gen -o countries_data.go

import "strings"

// Region is a UN M49 top-level region.
type Region string

const (
	RegionAfrica   Region = "Africa"
	RegionAmericas Region = "Americas"
	RegionAsia     Region = "Asia"
	RegionEurope   Region = "Europe"
	RegionOceania  Region = "Oceania"
)

// Country is one ISO 3166-1 entry and the groupings it belongs to.
//
// Region and SubRegion carry the UN M49 classification, which does not cover
// every ISO country: Antarctica and Taiwan have no region assigned, and for
// them both fields are empty. Treat them as unknown rather than absent.
type Country struct {
	// Name is the ISO 3166-1 English short name, which is the formal
	// spelling ("United Kingdom of Great Britain and Northern Ireland").
	// CommonName holds the everyday one.
	Name string

	// Alpha2 is the two-letter code, and the canonical identity here.
	Alpha2 string

	// Alpha3 is the three-letter code.
	Alpha3 string

	// Numeric is the three-digit code, zero-padded ("250", "004").
	Numeric string

	Region    Region
	SubRegion string

	// EU reports membership of the European Union.
	EU bool

	// EEA reports membership of the European Economic Area: the EU plus
	// Iceland, Liechtenstein and Norway.
	EEA bool
}

// CommonName is the name people actually use, which for most countries is the
// ISO short name and for a few is not ("France", but "United Kingdom" rather
// than "United Kingdom of Great Britain and Northern Ireland").
func (c Country) CommonName() string {
	if n, ok := commonNames[c.Alpha2]; ok {
		return n
	}
	return c.Name
}

// String returns the alpha-2 code, so a Country prints as the code that
// identifies it.
func (c Country) String() string {
	return c.Alpha2
}

// Get returns the country for a code, which may be alpha-2, alpha-3 or
// numeric, in any case. It reports ok=false for a code it does not know,
// leaving the fallback to the caller.
func Get(code string) (Country, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	switch len(code) {
	case 2:
		c, ok := byAlpha2[code]
		return c, ok
	case 3:
		if c, ok := byAlpha3[code]; ok {
			return c, true
		}
		c, ok := byNumeric[code]
		return c, ok
	}
	return Country{}, false
}

// Valid reports whether code names an ISO 3166-1 country, in any of the three
// spellings.
func Valid(code string) bool {
	_, ok := Get(code)
	return ok
}

// All returns every country, ordered by alpha-2 code. The slice is a fresh
// copy: callers may sort or filter it without disturbing the package tables.
func All() []Country {
	out := make([]Country, len(all))
	copy(out, all)
	return out
}

// InRegion returns the countries of a UN M49 region, ordered by alpha-2 code.
func InRegion(r Region) []Country {
	return filter(func(c Country) bool { return c.Region == r })
}

// InSubRegion returns the countries of a UN M49 sub-region ("Western Europe"),
// ordered by alpha-2 code. The match is case-insensitive.
func InSubRegion(sub string) []Country {
	sub = strings.TrimSpace(sub)
	return filter(func(c Country) bool { return strings.EqualFold(c.SubRegion, sub) })
}

// EUMembers returns the European Union member states, ordered by alpha-2 code.
func EUMembers() []Country {
	return filter(func(c Country) bool { return c.EU })
}

// EEAMembers returns the European Economic Area member states, ordered by
// alpha-2 code.
func EEAMembers() []Country {
	return filter(func(c Country) bool { return c.EEA })
}

// IsEU reports whether code names an EU member state. An unknown code is not
// a member.
func IsEU(code string) bool {
	c, ok := Get(code)
	return ok && c.EU
}

// IsEEA reports whether code names an EEA member state. An unknown code is
// not a member.
func IsEEA(code string) bool {
	c, ok := Get(code)
	return ok && c.EEA
}

func filter(keep func(Country) bool) []Country {
	var out []Country
	for _, c := range all {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}
