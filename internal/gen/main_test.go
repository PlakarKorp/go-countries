package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// header is the column line of the upstream CSV. The generator reads columns
// by name, so the tests spell the real header once and vary the rows.
const header = "name,alpha-2,alpha-3,country-code,iso_3166-2,region,sub-region," +
	"intermediate-region,region-code,sub-region-code,intermediate-region-code"

// csvOf builds a source document from rows given as full CSV lines.
func csvOf(rows ...string) []byte {
	return []byte(header + "\n" + strings.Join(rows, "\n") + "\n")
}

// row spells one country the way the upstream file does. The columns the
// generator ignores are left empty.
func row(name, a2, a3, numeric, region, subRegion string) string {
	return strings.Join([]string{
		name, a2, a3, numeric, "ISO 3166-2:" + a2, region, subRegion, "", "", "", "",
	}, ",")
}

// minimal is a source holding every EU and EEA member plus one country that is
// in neither, which is the least that satisfies the membership check.
func minimal(extra ...string) []byte {
	var rows []string
	for _, code := range euMembers {
		rows = append(rows, row("Country "+code, code, code+"X", "001", "Europe", "Western Europe"))
	}
	for _, code := range eftaEEA {
		rows = append(rows, row("Country "+code, code, code+"X", "002", "Europe", "Northern Europe"))
	}
	rows = append(rows, row("Testland", "ZZ", "ZZZ", "999", "Asia", "Western Asia"))
	return csvOf(append(rows, extra...)...)
}

func TestParseSortsByAlpha2(t *testing.T) {
	data := csvOf(
		row("Zed", "ZW", "ZWE", "716", "Africa", "Eastern Africa"),
		row("Alpha", "AL", "ALB", "008", "Europe", "Southern Europe"),
	)
	// The membership check needs the members present, so this source is only
	// used to look at ordering, and parse is expected to reject it.
	if _, err := parse(data); err == nil {
		t.Fatal("expected the membership check to reject a source without EU members")
	}

	got, err := parse(minimal(
		row("Zed", "ZW", "ZWE", "716", "Africa", "Eastern Africa"),
		row("Alpha", "AL", "ALB", "008", "Europe", "Southern Europe"),
	))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].alpha2 >= got[i].alpha2 {
			t.Fatalf("out of order at %d: %q then %q", i, got[i-1].alpha2, got[i].alpha2)
		}
	}
}

func TestParseAssignsMembership(t *testing.T) {
	got, err := parse(minimal())
	if err != nil {
		t.Fatal(err)
	}

	by := make(map[string]country, len(got))
	for _, c := range got {
		by[c.alpha2] = c
	}

	// An EU member is in the EEA by construction, an EFTA member is in the
	// EEA but not the EU, and a country in neither has both flags clear.
	if c := by["FR"]; !c.eu || !c.eea {
		t.Errorf("FR = {eu:%v eea:%v}, want both true", c.eu, c.eea)
	}
	if c := by["NO"]; c.eu || !c.eea {
		t.Errorf("NO = {eu:%v eea:%v}, want {false true}", c.eu, c.eea)
	}
	if c := by["ZZ"]; c.eu || c.eea {
		t.Errorf("ZZ = {eu:%v eea:%v}, want both false", c.eu, c.eea)
	}
}

// The membership lists are the generator's own data, so their sizes are worth
// asserting here as well as in the package tests: the counts the generator
// checks are the ones it writes.
func TestMembershipListsAreTheRightSize(t *testing.T) {
	if len(set(euMembers)) != 27 {
		t.Errorf("euMembers has %d distinct entries, want 27", len(set(euMembers)))
	}
	if len(set(eftaEEA)) != 3 {
		t.Errorf("eftaEEA has %d distinct entries, want 3", len(set(eftaEEA)))
	}
	// Switzerland is in EFTA but not the EEA, which is the entry most easily
	// added by mistake.
	if set(eftaEEA)["CH"] || set(euMembers)["CH"] {
		t.Error("CH is listed as an EEA or EU member")
	}
	for _, code := range eftaEEA {
		if set(euMembers)[code] {
			t.Errorf("%s is in both euMembers and eftaEEA", code)
		}
	}
}

func TestParseFields(t *testing.T) {
	got, err := parse(minimal(row("Testonia", "TO", "TON", "042", "Oceania", "Polynesia")))
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, c := range got {
		if c.alpha2 != "TO" {
			continue
		}
		found = true
		want := country{
			name: "Testonia", alpha2: "TO", alpha3: "TON", numeric: "042",
			region: "Oceania", subRegion: "Polynesia",
		}
		if c != want {
			t.Errorf("TO = %+v, want %+v", c, want)
		}
	}
	if !found {
		t.Error("TO missing from the parsed table")
	}
}

// An unclassified country keeps empty region fields rather than being dropped,
// since that is how Antarctica and Taiwan reach the table.
func TestParseKeepsUnclassifiedCountries(t *testing.T) {
	got, err := parse(minimal(row("Nowhere", "NW", "NWH", "010", "", "")))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c.alpha2 == "NW" {
			if c.region != "" || c.subRegion != "" {
				t.Errorf("NW = {region:%q sub:%q}, want both empty", c.region, c.subRegion)
			}
			return
		}
	}
	t.Error("NW was dropped")
}

// A row with no alpha-2 code is skipped rather than becoming a country with an
// empty identity.
func TestParseSkipsRowsWithoutAnAlpha2(t *testing.T) {
	got, err := parse(minimal(row("Nameless", "", "NON", "000", "Europe", "Western Europe")))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c.alpha2 == "" {
			t.Fatal("a row without an alpha-2 code became a country")
		}
	}
}

func TestParseTrimsWhitespace(t *testing.T) {
	got, err := parse(minimal(row("  Spacey  ", "  SP  ", "  SPC  ", "  123  ", "  Asia  ", "  Western Asia  ")))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c.alpha2 != "SP" {
			continue
		}
		want := country{
			name: "Spacey", alpha2: "SP", alpha3: "SPC", numeric: "123",
			region: "Asia", subRegion: "Western Asia",
		}
		if c != want {
			t.Errorf("SP = %+v, want %+v", c, want)
		}
		return
	}
	t.Error("SP missing; the alpha-2 code was not trimmed")
}

// The generator refuses a source it cannot vouch for. Each of these would
// otherwise produce a table that fails the package tests, or worse, a table
// that passes them while being quietly wrong.
func TestParseRejectsBadSources(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "missing column",
			data: []byte(strings.Replace(header, "alpha-3", "alpha3", 1) + "\n" +
				row("France", "FR", "FRA", "250", "Europe", "Western Europe") + "\n"),
			want: `no "alpha-3" column`,
		},
		{
			name: "header only",
			data: []byte(header + "\n"),
			want: "want a header and data",
		},
		{
			name: "empty",
			data: []byte(""),
			want: "want a header and data",
		},
		{
			name: "unknown region",
			data: minimal(row("Atlantean", "AT2", "ATL", "000", "Atlantis", "Deep")),
			want: `unknown region "Atlantis"`,
		},
		{
			name: "duplicate country",
			data: minimal(row("France Again", "FR", "FRZ", "251", "Europe", "Western Europe")),
			want: "FR appears twice",
		},
		{
			name: "membership names a country the source lacks",
			data: csvOf(row("Testland", "ZZ", "ZZZ", "999", "Asia", "Western Asia")),
			want: "which the source does not list",
		},
		{
			name: "ragged rows",
			data: []byte(header + "\nFrance,FR\n"),
			want: "wrong number of fields",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parse(tc.data)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

// render must emit compilable Go that names the region constants rather than
// repeating string literals, and must omit the membership fields when they are
// false so the table stays readable.
func TestRenderEmitsTheExpectedSource(t *testing.T) {
	src, err := render([]country{
		{name: "France", alpha2: "FR", alpha3: "FRA", numeric: "250",
			region: "Europe", subRegion: "Western Europe", eu: true, eea: true},
		{name: "Norway", alpha2: "NO", alpha3: "NOR", numeric: "578",
			region: "Europe", subRegion: "Northern Europe", eea: true},
		{name: "Antarctica", alpha2: "AQ", alpha3: "ATA", numeric: "010"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(src)

	for _, want := range []string{
		"// Code generated by internal/gen; DO NOT EDIT.",
		"package countries",
		"var all = []Country{",
		`{Name: "France", Alpha2: "FR", Alpha3: "FRA", Numeric: "250", Region: RegionEurope, SubRegion: "Western Europe", EU: true, EEA: true},`,
		`{Name: "Norway", Alpha2: "NO", Alpha3: "NOR", Numeric: "578", Region: RegionEurope, SubRegion: "Northern Europe", EEA: true},`,
		`{Name: "Antarctica", Alpha2: "AQ", Alpha3: "ATA", Numeric: "010", Region: "", SubRegion: ""},`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain:\n%s\ngot:\n%s", want, got)
		}
	}

	// Norway is not an EU member, so the field must be absent rather than
	// spelled out as false.
	if strings.Contains(got, `"Norway", Alpha2: "NO", Alpha3: "NOR", Numeric: "578", Region: RegionEurope, SubRegion: "Northern Europe", EU:`) {
		t.Error("Norway carries an EU field")
	}
}

// render goes through go/format, so a name carrying a quote or a backslash has
// to survive as a valid literal rather than breaking the file open.
func TestRenderQuotesAwkwardNames(t *testing.T) {
	src, err := render([]country{
		{name: `Côte d'Ivoire`, alpha2: "CI", alpha3: "CIV", numeric: "384",
			region: "Africa", subRegion: "Western Africa"},
		{name: `The "Quoted" \ Republic`, alpha2: "QQ", alpha3: "QQQ", numeric: "001",
			region: "Asia", subRegion: "Western Asia"},
	})
	if err != nil {
		t.Fatalf("render produced unformattable Go: %v", err)
	}
	got := string(src)
	if !strings.Contains(got, `Name: "Côte d'Ivoire"`) {
		t.Error("the accented name did not survive")
	}
	if !strings.Contains(got, `Name: "The \"Quoted\" \\ Republic"`) {
		t.Errorf("the quoted name was not escaped:\n%s", got)
	}
}

// The output is formatted, so it is gofmt-clean by construction. go/format
// returning an error is the failure this guards, since it means render emitted
// something that is not Go at all.
func TestRenderOutputIsFormatted(t *testing.T) {
	src, err := render([]country{
		{name: "France", alpha2: "FR", alpha3: "FRA", numeric: "250",
			region: "Europe", subRegion: "Western Europe", eu: true, eea: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(src), "}\n") {
		t.Errorf("output does not end in a closed brace and newline:\n%s", src)
	}
}

func TestRenderEmptyTable(t *testing.T) {
	src, err := render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "var all = []Country{}") &&
		!strings.Contains(string(src), "var all = []Country{\n}") {
		t.Errorf("unexpected empty table:\n%s", src)
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "all.csv")
	want := minimal()
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := read(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("read returned different bytes than were written")
	}
}

func TestReadMissingFile(t *testing.T) {
	if _, err := read(filepath.Join(t.TempDir(), "absent.csv")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestReadURL(t *testing.T) {
	want := minimal()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(want)
	}))
	defer srv.Close()

	got, err := read(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("read returned different bytes than the server sent")
	}
}

// A store that answers with an error page must not be mistaken for data: the
// status is checked before the body is read.
func TestReadURLBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := read(srv.URL)
	if err == nil {
		t.Fatal("expected an error for a 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q does not mention the status", err)
	}
}

func TestReadURLUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if _, err := read(url); err == nil {
		t.Fatal("expected an error for an unreachable server")
	}
}

// generate is the whole pipeline, so this is the test that would catch a
// regression in how the pieces are wired together.
func TestGenerateWritesATable(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "all.csv")
	if err := os.WriteFile(source, minimal(), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "countries_data.go")

	n, err := generate(source, out)
	if err != nil {
		t.Fatal(err)
	}
	if want := len(euMembers) + len(eftaEEA) + 1; n != want {
		t.Errorf("wrote %d countries, want %d", n, want)
	}

	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "package countries") {
		t.Errorf("output is not the generated package:\n%s", written)
	}
}

func TestGenerateReportsFailures(t *testing.T) {
	dir := t.TempDir()

	t.Run("unreadable source", func(t *testing.T) {
		if _, err := generate(filepath.Join(dir, "absent.csv"), filepath.Join(dir, "out.go")); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("bad source", func(t *testing.T) {
		source := filepath.Join(dir, "bad.csv")
		if err := os.WriteFile(source, []byte(header+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := generate(source, filepath.Join(dir, "out.go")); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("unwritable destination", func(t *testing.T) {
		source := filepath.Join(dir, "good.csv")
		if err := os.WriteFile(source, minimal(), 0o644); err != nil {
			t.Fatal(err)
		}
		// A directory that does not exist cannot be written into.
		out := filepath.Join(dir, "absent-dir", "out.go")
		if _, err := generate(source, out); err == nil {
			t.Fatal("expected an error")
		}
	})
}

// regionConsts must name a constant for every region the source can carry, and
// parse rejects anything else, so the two have to agree.
func TestRegionConstsCoverTheM49Regions(t *testing.T) {
	for _, region := range []string{"Africa", "Americas", "Asia", "Europe", "Oceania"} {
		if _, ok := regionConsts[region]; !ok {
			t.Errorf("regionConsts has no entry for %q", region)
		}
	}
	if len(regionConsts) != 5 {
		t.Errorf("regionConsts has %d entries, want 5", len(regionConsts))
	}
}
