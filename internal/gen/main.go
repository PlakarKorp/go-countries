// Command gen writes countries_data.go from the ISO 3166-1 list published
// with UN M49 region codes.
//
// It reads the CSV from a local path or, by default, from the upstream URL,
// and formats its output with go/format so the result needs no gofmt pass:
//
//	go run ./internal/gen -o countries_data.go
//	go run ./internal/gen -source all.csv -o countries_data.go
//
// EU and EEA membership are political facts the CSV does not carry, so they
// are maintained here. The package tests assert their cardinality, which is
// what keeps a bad edit from shipping quietly.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

const sourceURL = "https://raw.githubusercontent.com/lukes/ISO-3166-Countries-with-Regional-Codes/master/all/all.csv"

// euMembers is the 27 member states of the European Union.
var euMembers = []string{
	"AT", "BE", "BG", "CY", "CZ", "DE", "DK", "EE", "ES", "FI", "FR", "GR",
	"HR", "HU", "IE", "IT", "LT", "LU", "LV", "MT", "NL", "PL", "PT", "RO",
	"SE", "SI", "SK",
}

// eftaEEA is the EEA members that are not in the EU. Switzerland is in EFTA
// but not the EEA, so it is deliberately absent.
var eftaEEA = []string{"IS", "LI", "NO"}

// regionConsts maps a UN M49 region name to the package constant naming it,
// so the generated table reads in terms of the exported vocabulary rather
// than repeating string literals.
var regionConsts = map[string]string{
	"Africa":   "RegionAfrica",
	"Americas": "RegionAmericas",
	"Asia":     "RegionAsia",
	"Europe":   "RegionEurope",
	"Oceania":  "RegionOceania",
}

type country struct {
	name      string
	alpha2    string
	alpha3    string
	numeric   string
	region    string
	subRegion string
	eu        bool
	eea       bool
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("gen: ")

	source := flag.String("source", sourceURL, "CSV path or URL to read the ISO 3166-1 list from")
	out := flag.String("o", "countries_data.go", "file to write")
	flag.Parse()

	n, err := generate(*source, *out)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d countries to %s\n", n, *out)
}

// generate reads source, renders the table and writes it to out, returning the
// number of countries written. main is left holding nothing but the flags so
// that the whole pipeline is reachable from a test.
func generate(source, out string) (int, error) {
	data, err := read(source)
	if err != nil {
		return 0, err
	}

	countries, err := parse(data)
	if err != nil {
		return 0, err
	}

	src, err := render(countries)
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(out, src, 0o644); err != nil {
		return 0, err
	}
	return len(countries), nil
}

// read returns the bytes of source, which is a URL when it looks like one and
// a file otherwise.
func read(source string) ([]byte, error) {
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		return os.ReadFile(source)
	}

	resp, err := http.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: %s", source, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// parse turns the CSV into countries sorted by alpha-2 code, which is the
// order the package promises. It fails rather than emitting a table that
// would not survive the package tests.
func parse(data []byte) ([]country, error) {
	records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("source has %d rows, want a header and data", len(records))
	}

	columns := make(map[string]int, len(records[0]))
	for i, name := range records[0] {
		columns[name] = i
	}
	for _, want := range []string{"name", "alpha-2", "alpha-3", "country-code", "region", "sub-region"} {
		if _, ok := columns[want]; !ok {
			return nil, fmt.Errorf("source has no %q column", want)
		}
	}
	field := func(row []string, name string) string {
		return strings.TrimSpace(row[columns[name]])
	}

	eu := set(euMembers)
	if len(eu) != 27 {
		return nil, fmt.Errorf("euMembers has %d entries, want 27", len(eu))
	}
	eea := set(eftaEEA)
	if len(eea) != 3 {
		return nil, fmt.Errorf("eftaEEA has %d entries, want 3", len(eea))
	}

	var countries []country
	seen := make(map[string]bool)
	for _, row := range records[1:] {
		c := country{
			name:      field(row, "name"),
			alpha2:    field(row, "alpha-2"),
			alpha3:    field(row, "alpha-3"),
			numeric:   field(row, "country-code"),
			region:    field(row, "region"),
			subRegion: field(row, "sub-region"),
		}
		if c.alpha2 == "" {
			continue
		}
		if seen[c.alpha2] {
			return nil, fmt.Errorf("%s appears twice in the source", c.alpha2)
		}
		seen[c.alpha2] = true

		if c.region != "" {
			if _, ok := regionConsts[c.region]; !ok {
				return nil, fmt.Errorf("%s has unknown region %q", c.alpha2, c.region)
			}
		}

		c.eu = eu[c.alpha2]
		c.eea = c.eu || eea[c.alpha2]
		countries = append(countries, c)
	}

	// A membership entry naming a country the source does not have is a typo,
	// and would otherwise cost the table a member without saying so.
	for _, code := range append(append([]string{}, euMembers...), eftaEEA...) {
		if !seen[code] {
			return nil, fmt.Errorf("membership names %q, which the source does not list", code)
		}
	}

	sort.Slice(countries, func(i, j int) bool {
		return countries[i].alpha2 < countries[j].alpha2
	})
	return countries, nil
}

func set(codes []string) map[string]bool {
	m := make(map[string]bool, len(codes))
	for _, c := range codes {
		m[c] = true
	}
	return m
}

func render(countries []country) ([]byte, error) {
	var b strings.Builder
	b.WriteString("// Code generated by internal/gen; DO NOT EDIT.\n\n")
	b.WriteString("package countries\n\n")
	b.WriteString("// all is every ISO 3166-1 country, ordered by alpha-2 code.\n")
	b.WriteString("var all = []Country{\n")

	for _, c := range countries {
		region := `""`
		if c.region != "" {
			region = regionConsts[c.region]
		}

		fmt.Fprintf(&b, "\t{Name: %q, Alpha2: %q, Alpha3: %q, Numeric: %q, Region: %s, SubRegion: %q",
			c.name, c.alpha2, c.alpha3, c.numeric, region, c.subRegion)
		if c.eu {
			b.WriteString(", EU: true")
		}
		if c.eea {
			b.WriteString(", EEA: true")
		}
		b.WriteString("},\n")
	}

	b.WriteString("}\n")
	return format.Source([]byte(b.String()))
}
