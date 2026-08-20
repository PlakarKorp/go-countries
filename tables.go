package countries

// The lookup tables are built once, at init, from the generated slice. Doing
// it here rather than in the generated file keeps the generated output to the
// data itself, and keeps the three indexes consistent by construction.
var (
	byAlpha2  = make(map[string]Country, len(all))
	byAlpha3  = make(map[string]Country, len(all))
	byNumeric = make(map[string]Country, len(all))
)

func init() {
	for _, c := range all {
		byAlpha2[c.Alpha2] = c
		byAlpha3[c.Alpha3] = c
		byNumeric[c.Numeric] = c
	}
}

// commonNames overrides the ISO short name where the formal spelling is not
// what anybody writes. Countries absent from this table use their ISO name.
var commonNames = map[string]string{
	"BO": "Bolivia",
	"BN": "Brunei",
	"CD": "Democratic Republic of the Congo",
	"CI": "Ivory Coast",
	"CV": "Cape Verde",
	"FK": "Falkland Islands",
	"FM": "Micronesia",
	"GB": "United Kingdom",
	"IR": "Iran",
	"KP": "North Korea",
	"KR": "South Korea",
	"LA": "Laos",
	"MD": "Moldova",
	"PS": "Palestine",
	"RU": "Russia",
	"SY": "Syria",
	"TW": "Taiwan",
	"TZ": "Tanzania",
	"US": "United States",
	"VA": "Vatican City",
	"VE": "Venezuela",
	"VN": "Vietnam",
}
