package countries

import "testing"

// The table is the whole point of the package, so the tests spend most of
// their effort on its integrity rather than on the lookup wrappers.

func TestTableIsComplete(t *testing.T) {
	// ISO 3166-1 has 249 assigned entries; a change here is either an ISO
	// revision or a regeneration accident, and both deserve a look.
	if len(all) != 249 {
		t.Errorf("got %d countries, want 249", len(all))
	}
}

func TestCodesAreWellFormed(t *testing.T) {
	for _, c := range all {
		if len(c.Alpha2) != 2 {
			t.Errorf("%s: alpha-2 %q is not two letters", c.Name, c.Alpha2)
		}
		if len(c.Alpha3) != 3 {
			t.Errorf("%s: alpha-3 %q is not three letters", c.Name, c.Alpha3)
		}
		if len(c.Numeric) != 3 {
			t.Errorf("%s: numeric %q is not three digits", c.Name, c.Numeric)
		}
		for _, r := range c.Numeric {
			if r < '0' || r > '9' {
				t.Errorf("%s: numeric %q is not all digits", c.Name, c.Numeric)
				break
			}
		}
		if c.Name == "" {
			t.Errorf("%s: empty name", c.Alpha2)
		}
	}
}

func TestCodesAreUnique(t *testing.T) {
	// A duplicate would be invisible through the maps, which would simply
	// keep the last entry, so the slice is what gets counted.
	for name, index := range map[string]map[string]string{
		"alpha-2": {},
		"alpha-3": {},
		"numeric": {},
	} {
		for _, c := range all {
			code := map[string]string{"alpha-2": c.Alpha2, "alpha-3": c.Alpha3, "numeric": c.Numeric}[name]
			if prev, dup := index[code]; dup {
				t.Errorf("%s %q used by both %s and %s", name, code, prev, c.Alpha2)
			}
			index[code] = c.Alpha2
		}
	}
}

func TestTableIsSortedByAlpha2(t *testing.T) {
	// All() and the filters promise alpha-2 order, and they get it from the
	// generated slice rather than sorting at call time.
	for i := 1; i < len(all); i++ {
		if all[i-1].Alpha2 >= all[i].Alpha2 {
			t.Fatalf("out of order at %d: %q then %q", i, all[i-1].Alpha2, all[i].Alpha2)
		}
	}
}

func TestIndexesCoverTheTable(t *testing.T) {
	if len(byAlpha2) != len(all) || len(byAlpha3) != len(all) || len(byNumeric) != len(all) {
		t.Fatalf("index sizes %d/%d/%d, want %d each",
			len(byAlpha2), len(byAlpha3), len(byNumeric), len(all))
	}
}

func TestGetAcceptsAllThreeSpellings(t *testing.T) {
	for _, code := range []string{"FR", "FRA", "250", "fr", "fra", " FR "} {
		c, ok := Get(code)
		if !ok {
			t.Errorf("Get(%q) failed", code)
			continue
		}
		if c.Alpha2 != "FR" {
			t.Errorf("Get(%q) = %s, want FR", code, c.Alpha2)
		}
	}
}

func TestGetRejectsUnknown(t *testing.T) {
	// "XX" is unassigned, "ZZZ" is not a code, and the empty and overlong
	// strings exercise the length switch's default.
	for _, code := range []string{"", "X", "XX", "ZZZ", "999", "FRANCE"} {
		if c, ok := Get(code); ok {
			t.Errorf("Get(%q) unexpectedly returned %s", code, c.Alpha2)
		}
	}
}

// Numeric codes are three characters wide, which is also alpha-3's width, so
// Get resolves the two against separate tables. A numeric code must not be
// shadowed by the alpha-3 lookup that is tried first.
func TestNumericAndAlpha3DoNotCollide(t *testing.T) {
	for _, c := range all {
		if got, ok := Get(c.Numeric); !ok || got.Alpha2 != c.Alpha2 {
			t.Errorf("Get(%q) did not resolve to %s", c.Numeric, c.Alpha2)
		}
		if got, ok := Get(c.Alpha3); !ok || got.Alpha2 != c.Alpha2 {
			t.Errorf("Get(%q) did not resolve to %s", c.Alpha3, c.Alpha2)
		}
	}
}

func TestValid(t *testing.T) {
	for _, code := range []string{"FR", "usa", "840", "AQ"} {
		if !Valid(code) {
			t.Errorf("Valid(%q) = false", code)
		}
	}
	for _, code := range []string{"XX", "", "EU"} {
		if Valid(code) {
			t.Errorf("Valid(%q) = true", code)
		}
	}
}

func TestAllReturnsACopy(t *testing.T) {
	got := All()
	if len(got) != len(all) {
		t.Fatalf("All() returned %d, want %d", len(got), len(all))
	}
	got[0].Name = "clobbered"
	if all[0].Name == "clobbered" {
		t.Error("All() shares memory with the package table")
	}
}

func TestEUMembership(t *testing.T) {
	// The EU has 27 member states post-Brexit, and the UK must not be one.
	eu := EUMembers()
	if len(eu) != 27 {
		t.Errorf("got %d EU members, want 27", len(eu))
	}
	for _, code := range []string{"FR", "DE", "IE", "HR"} {
		if !IsEU(code) {
			t.Errorf("IsEU(%q) = false", code)
		}
	}
	for _, code := range []string{"GB", "CH", "NO", "US", "XX"} {
		if IsEU(code) {
			t.Errorf("IsEU(%q) = true", code)
		}
	}
}

func TestEEAMembership(t *testing.T) {
	// The EEA is the 27 plus Iceland, Liechtenstein and Norway.
	eea := EEAMembers()
	if len(eea) != 30 {
		t.Errorf("got %d EEA members, want 30", len(eea))
	}
	for _, code := range []string{"FR", "IS", "LI", "NO"} {
		if !IsEEA(code) {
			t.Errorf("IsEEA(%q) = false", code)
		}
	}
	// Switzerland is in EFTA but not the EEA, which is the distinction most
	// often got wrong.
	for _, code := range []string{"CH", "GB", "US"} {
		if IsEEA(code) {
			t.Errorf("IsEEA(%q) = true", code)
		}
	}
}

// Every EU member is in the EEA, so the flags cannot disagree in that
// direction however the data is regenerated.
func TestEUImpliesEEA(t *testing.T) {
	for _, c := range all {
		if c.EU && !c.EEA {
			t.Errorf("%s is EU but not EEA", c.Alpha2)
		}
	}
}

func TestRegions(t *testing.T) {
	if got := len(InRegion(RegionEurope)); got == 0 {
		t.Error("InRegion(Europe) is empty")
	}
	if got := InRegion(Region("Atlantis")); got != nil {
		t.Errorf("InRegion(Atlantis) = %v, want nil", got)
	}
	for _, c := range InRegion(RegionEurope) {
		if c.Region != RegionEurope {
			t.Errorf("%s has region %q in the Europe set", c.Alpha2, c.Region)
		}
	}
}

// UN M49 does not classify every ISO country. The two it omits are known, and
// a change in that set means the data source shifted under us.
func TestOnlyKnownCountriesLackARegion(t *testing.T) {
	unclassified := map[string]bool{"AQ": true, "TW": true}
	for _, c := range all {
		if c.Region == "" && !unclassified[c.Alpha2] {
			t.Errorf("%s (%s) has no region", c.Alpha2, c.Name)
		}
		if c.Region != "" && unclassified[c.Alpha2] {
			t.Errorf("%s now has region %q; update the expectation", c.Alpha2, c.Region)
		}
	}
}

func TestRegionsAreFromTheKnownSet(t *testing.T) {
	known := map[Region]bool{
		RegionAfrica: true, RegionAmericas: true, RegionAsia: true,
		RegionEurope: true, RegionOceania: true, "": true,
	}
	for _, c := range all {
		if !known[c.Region] {
			t.Errorf("%s has unexpected region %q", c.Alpha2, c.Region)
		}
	}
}

func TestInSubRegion(t *testing.T) {
	west := InSubRegion("western europe")
	if len(west) == 0 {
		t.Fatal("InSubRegion is case-sensitive or empty")
	}
	var sawFR bool
	for _, c := range west {
		if c.Alpha2 == "FR" {
			sawFR = true
		}
	}
	if !sawFR {
		t.Error("Western Europe does not contain FR")
	}
	if got := InSubRegion("Nowhere"); got != nil {
		t.Errorf("InSubRegion(Nowhere) = %v, want nil", got)
	}
}

func TestCommonNameOverridesAreLive(t *testing.T) {
	// An override keyed on a bad code, or one that merely repeats the ISO
	// name, is dead weight that no other test would notice.
	for code, name := range commonNames {
		c, ok := byAlpha2[code]
		if !ok {
			t.Errorf("commonNames has unknown code %q", code)
			continue
		}
		if c.Name == name {
			t.Errorf("%s: override %q repeats the ISO name", code, name)
		}
		if got := c.CommonName(); got != name {
			t.Errorf("%s: CommonName() = %q, want %q", code, got, name)
		}
	}
}

func TestCommonNameFallsBackToISOName(t *testing.T) {
	fr, _ := Get("FR")
	if got := fr.CommonName(); got != "France" {
		t.Errorf("FR CommonName() = %q, want France", got)
	}
	gb, _ := Get("GB")
	if got := gb.CommonName(); got != "United Kingdom" {
		t.Errorf("GB CommonName() = %q, want United Kingdom", got)
	}
}

func TestStringIsTheAlpha2Code(t *testing.T) {
	c, _ := Get("DEU")
	if got := c.String(); got != "DE" {
		t.Errorf("String() = %q, want DE", got)
	}
}

func TestKnownEntries(t *testing.T) {
	for _, want := range []Country{
		{Name: "France", Alpha2: "FR", Alpha3: "FRA", Numeric: "250", Region: RegionEurope, SubRegion: "Western Europe", EU: true, EEA: true},
		{Name: "United States of America", Alpha2: "US", Alpha3: "USA", Numeric: "840", Region: RegionAmericas, SubRegion: "Northern America"},
		{Name: "Antarctica", Alpha2: "AQ", Alpha3: "ATA", Numeric: "010"},
	} {
		got, ok := Get(want.Alpha2)
		if !ok {
			t.Errorf("%s missing", want.Alpha2)
			continue
		}
		if got != want {
			// Country has a String method, so %+v on it would print the
			// alpha-2 code and hide the field that actually differs.
			type fields Country
			t.Errorf("%s = %+v, want %+v", want.Alpha2, fields(got), fields(want))
		}
	}
}

// A country the UN classifies has both a region and a sub-region: the two
// fields come from the same row, so one arriving without the other means the
// generation slipped rather than that the data says something interesting.
func TestRegionAndSubRegionAgreeOnBeingKnown(t *testing.T) {
	for _, c := range all {
		if (c.Region == "") != (c.SubRegion == "") {
			t.Errorf("%s: region %q but sub-region %q", c.Alpha2, c.Region, c.SubRegion)
		}
	}
}

// A sub-region belongs to exactly one region. Nothing in the row format
// enforces that, so a shifted column would show up here as "Western Europe"
// turning up under two different regions.
func TestSubRegionsNestUnderOneRegion(t *testing.T) {
	parent := make(map[string]Region)
	for _, c := range all {
		if c.SubRegion == "" {
			continue
		}
		if got, seen := parent[c.SubRegion]; seen && got != c.Region {
			t.Errorf("sub-region %q appears under both %q and %q",
				c.SubRegion, got, c.Region)
			continue
		}
		parent[c.SubRegion] = c.Region
	}
	if len(parent) == 0 {
		t.Fatal("no sub-regions in the table")
	}
}

// The five regions partition every classified country, so their sizes have to
// add up to the table minus the two the UN leaves out. A regeneration that
// dropped or duplicated rows fails here even when each surviving row is
// individually well-formed.
func TestRegionsPartitionTheTable(t *testing.T) {
	regions := []Region{RegionAfrica, RegionAmericas, RegionAsia, RegionEurope, RegionOceania}

	var total int
	for _, r := range regions {
		in := InRegion(r)
		if len(in) == 0 {
			t.Errorf("InRegion(%q) is empty", r)
		}
		for _, c := range in {
			if c.Region != r {
				t.Errorf("%s has region %q in the %q set", c.Alpha2, c.Region, r)
			}
		}
		total += len(in)
	}

	unclassified := len(filter(func(c Country) bool { return c.Region == "" }))
	if total+unclassified != len(all) {
		t.Errorf("regions cover %d + %d unclassified, want %d",
			total, unclassified, len(all))
	}
}

// InSubRegion must return every country of a sub-region and nothing else, so
// the filter is checked against the table rather than against a spot-check.
func TestInSubRegionIsExhaustive(t *testing.T) {
	want := make(map[string][]string)
	for _, c := range all {
		if c.SubRegion != "" {
			want[c.SubRegion] = append(want[c.SubRegion], c.Alpha2)
		}
	}

	for sub, codes := range want {
		got := InSubRegion(sub)
		if len(got) != len(codes) {
			t.Errorf("InSubRegion(%q) returned %d, want %d", sub, len(got), len(codes))
			continue
		}
		for i, c := range got {
			if c.Alpha2 != codes[i] {
				t.Errorf("InSubRegion(%q)[%d] = %s, want %s", sub, i, c.Alpha2, codes[i])
			}
		}
	}
}

// Sub-region membership is data, and the counts are what a bad regeneration
// disturbs. These are the sizes the current ISO/M49 pairing yields; a change
// is either an ISO revision or an accident, and both want a human to look.
func TestSubRegionSizes(t *testing.T) {
	for sub, want := range map[string]int{
		"Western Europe":   9,
		"Northern Europe":  16,
		"Southern Europe":  16,
		"Eastern Europe":   10,
		"Northern America": 5,
	} {
		if got := len(InSubRegion(sub)); got != want {
			t.Errorf("%s has %d countries, want %d", sub, got, want)
		}
	}
}

// The EU sits inside Europe with one exception worth naming: Cyprus is a
// member state that M49 files under Western Asia. A residency rule written as
// "Europe" rather than "EU" gets Cyprus wrong, so the shape is pinned here.
func TestEUMembersAreEuropeanExceptCyprus(t *testing.T) {
	for _, c := range EUMembers() {
		if c.Region == RegionEurope {
			continue
		}
		if c.Alpha2 == "CY" {
			continue
		}
		t.Errorf("%s is EU but in region %q", c.Alpha2, c.Region)
	}
	cy, ok := Get("CY")
	if !ok {
		t.Fatal("CY missing")
	}
	if !cy.EU || cy.Region != RegionAsia {
		t.Errorf("CY = {EU:%v Region:%q}, want {true Asia}", cy.EU, cy.Region)
	}
}
