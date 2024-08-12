package services

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	LatitudeRegex = `^[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?)$`
	LongitudeRegex = `^[-+]?((1[0-7]\d)|([1-9]?\d))(\.\d+)?$`
)

type Country struct {
	Alpha3Code string
	Alpha2Code string
	Country    string
}

var Countries = map[string]Country{
	"AF": {Alpha3Code: "AFG", Alpha2Code: "AF", Country: "Afghanistan"},
	"AL": {Alpha3Code: "ALB", Alpha2Code: "AL", Country: "Albania"},
	"DZ": {Alpha3Code: "DZA", Alpha2Code: "DZ", Country: "Algeria"},
	"AS": {Alpha3Code: "ASM", Alpha2Code: "AS", Country: "American Samoa"},
	"AD": {Alpha3Code: "AND", Alpha2Code: "AD", Country: "Andorra"},
	"AO": {Alpha3Code: "AGO", Alpha2Code: "AO", Country: "Angola"},
	"AI": {Alpha3Code: "AIA", Alpha2Code: "AI", Country: "Anguilla"},
	"AQ": {Alpha3Code: "ATA", Alpha2Code: "AQ", Country: "Antarctica"},
	"AG": {Alpha3Code: "ATG", Alpha2Code: "AG", Country: "Antigua and Barbuda"},
	"AR": {Alpha3Code: "ARG", Alpha2Code: "AR", Country: "Argentina"},
	"AM": {Alpha3Code: "ARM", Alpha2Code: "AM", Country: "Armenia"},
	"AW": {Alpha3Code: "ABW", Alpha2Code: "AW", Country: "Aruba"},
	"AU": {Alpha3Code: "AUS", Alpha2Code: "AU", Country: "Australia"},
	"AT": {Alpha3Code: "AUT", Alpha2Code: "AT", Country: "Austria"},
	"AZ": {Alpha3Code: "AZE", Alpha2Code: "AZ", Country: "Azerbaijan"},
	"BS": {Alpha3Code: "BHS", Alpha2Code: "BS", Country: "Bahamas (the)"},
	"BH": {Alpha3Code: "BHR", Alpha2Code: "BH", Country: "Bahrain"},
	"BD": {Alpha3Code: "BGD", Alpha2Code: "BD", Country: "Bangladesh"},
	"BB": {Alpha3Code: "BRB", Alpha2Code: "BB", Country: "Barbados"},
	"BY": {Alpha3Code: "BLR", Alpha2Code: "BY", Country: "Belarus"},
	"BE": {Alpha3Code: "BEL", Alpha2Code: "BE", Country: "Belgium"},
	"BZ": {Alpha3Code: "BLZ", Alpha2Code: "BZ", Country: "Belize"},
	"BJ": {Alpha3Code: "BEN", Alpha2Code: "BJ", Country: "Benin"},
	"BM": {Alpha3Code: "BMU", Alpha2Code: "BM", Country: "Bermuda"},
	"BT": {Alpha3Code: "BTN", Alpha2Code: "BT", Country: "Bhutan"},
	"BO": {Alpha3Code: "BOL", Alpha2Code: "BO", Country: "Bolivia (Plurinational State of)"},
	"BQ": {Alpha3Code: "BES", Alpha2Code: "BQ", Country: "Bonaire, Sint Eustatius and Saba"},
	"BA": {Alpha3Code: "BIH", Alpha2Code: "BA", Country: "Bosnia and Herzegovina"},
	"BW": {Alpha3Code: "BWA", Alpha2Code: "BW", Country: "Botswana"},
	"BV": {Alpha3Code: "BVT", Alpha2Code: "BV", Country: "Bouvet Island"},
	"BR": {Alpha3Code: "BRA", Alpha2Code: "BR", Country: "Brazil"},
	"IO": {Alpha3Code: "IOT", Alpha2Code: "IO", Country: "British Indian Ocean Territory (the)"},
	"BN": {Alpha3Code: "BRN", Alpha2Code: "BN", Country: "Brunei Darussalam"},
	"BG": {Alpha3Code: "BGR", Alpha2Code: "BG", Country: "Bulgaria"},
	"BF": {Alpha3Code: "BFA", Alpha2Code: "BF", Country: "Burkina Faso"},
	"BI": {Alpha3Code: "BDI", Alpha2Code: "BI", Country: "Burundi"},
	"CV": {Alpha3Code: "CPV", Alpha2Code: "CV", Country: "Cabo Verde"},
	"KH": {Alpha3Code: "KHM", Alpha2Code: "KH", Country: "Cambodia"},
	"CM": {Alpha3Code: "CMR", Alpha2Code: "CM", Country: "Cameroon"},
	"CA": {Alpha3Code: "CAN", Alpha2Code: "CA", Country: "Canada"},
	"KY": {Alpha3Code: "CYM", Alpha2Code: "KY", Country: "Cayman Islands (the)"},
	"CF": {Alpha3Code: "CAF", Alpha2Code: "CF", Country: "Central African Republic (the)"},
	"TD": {Alpha3Code: "TCD", Alpha2Code: "TD", Country: "Chad"},
	"CL": {Alpha3Code: "CHL", Alpha2Code: "CL", Country: "Chile"},
	"CN": {Alpha3Code: "CHN", Alpha2Code: "CN", Country: "China"},
	"CX": {Alpha3Code: "CXR", Alpha2Code: "CX", Country: "Christmas Island"},
	"CC": {Alpha3Code: "CCK", Alpha2Code: "CC", Country: "Cocos (Keeling) Islands (the)"},
	"CO": {Alpha3Code: "COL", Alpha2Code: "CO", Country: "Colombia"},
	"KM": {Alpha3Code: "COM", Alpha2Code: "KM", Country: "Comoros (the)"},
	"CD": {Alpha3Code: "COD", Alpha2Code: "CD", Country: "Congo (the Democratic Republic of the)"},
	"CG": {Alpha3Code: "COG", Alpha2Code: "CG", Country: "Congo (the)"},
	"CK": {Alpha3Code: "COK", Alpha2Code: "CK", Country: "Cook Islands (the)"},
	"CR": {Alpha3Code: "CRI", Alpha2Code: "CR", Country: "Costa Rica"},
	"HR": {Alpha3Code: "HRV", Alpha2Code: "HR", Country: "Croatia"},
	"CU": {Alpha3Code: "CUB", Alpha2Code: "CU", Country: "Cuba"},
	"CW": {Alpha3Code: "CUW", Alpha2Code: "CW", Country: "Curaçao"},
	"CY": {Alpha3Code: "CYP", Alpha2Code: "CY", Country: "Cyprus"},
	"CZ": {Alpha3Code: "CZE", Alpha2Code: "CZ", Country: "Czechia"},
	"CI": {Alpha3Code: "CIV", Alpha2Code: "CI", Country: "Côte d'Ivoire"},
	"DK": {Alpha3Code: "DNK", Alpha2Code: "DK", Country: "Denmark"},
	"DJ": {Alpha3Code: "DJI", Alpha2Code: "DJ", Country: "Djibouti"},
	"DM": {Alpha3Code: "DMA", Alpha2Code: "DM", Country: "Dominica"},
	"DO": {Alpha3Code: "DOM", Alpha2Code: "DO", Country: "Dominican Republic (the)"},
	"EC": {Alpha3Code: "ECU", Alpha2Code: "EC", Country: "Ecuador"},
	"EG": {Alpha3Code: "EGY", Alpha2Code: "EG", Country: "Egypt"},
	"SV": {Alpha3Code: "SLV", Alpha2Code: "SV", Country: "El Salvador"},
	"GQ": {Alpha3Code: "GNQ", Alpha2Code: "GQ", Country: "Equatorial Guinea"},
	"ER": {Alpha3Code: "ERI", Alpha2Code: "ER", Country: "Eritrea"},
	"EE": {Alpha3Code: "EST", Alpha2Code: "EE", Country: "Estonia"},
	"SZ": {Alpha3Code: "SWZ", Alpha2Code: "SZ", Country: "Eswatini"},
	"ET": {Alpha3Code: "ETH", Alpha2Code: "ET", Country: "Ethiopia"},
	"FK": {Alpha3Code: "FLK", Alpha2Code: "FK", Country: "Falkland Islands (the) [Malvinas]"},
	"FO": {Alpha3Code: "FRO", Alpha2Code: "FO", Country: "Faroe Islands (the)"},
	"FJ": {Alpha3Code: "FJI", Alpha2Code: "FJ", Country: "Fiji"},
	"FI": {Alpha3Code: "FIN", Alpha2Code: "FI", Country: "Finland"},
	"FR": {Alpha3Code: "FRA", Alpha2Code: "FR", Country: "France"},
	"GF": {Alpha3Code: "GUF", Alpha2Code: "GF", Country: "French Guiana"},
	"PF": {Alpha3Code: "PYF", Alpha2Code: "PF", Country: "French Polynesia"},
	"TF": {Alpha3Code: "ATF", Alpha2Code: "TF", Country: "French Southern Territories (the)"},
	"GA": {Alpha3Code: "GAB", Alpha2Code: "GA", Country: "Gabon"},
	"GM": {Alpha3Code: "GMB", Alpha2Code: "GM", Country: "Gambia (the)"},
	"GE": {Alpha3Code: "GEO", Alpha2Code: "GE", Country: "Georgia"},
	"DE": {Alpha3Code: "DEU", Alpha2Code: "DE", Country: "Germany"},
	"GH": {Alpha3Code: "GHA", Alpha2Code: "GH", Country: "Ghana"},
	"GI": {Alpha3Code: "GIB", Alpha2Code: "GI", Country: "Gibraltar"},
	"GR": {Alpha3Code: "GRC", Alpha2Code: "GR", Country: "Greece"},
	"GL": {Alpha3Code: "GRL", Alpha2Code: "GL", Country: "Greenland"},
	"GD": {Alpha3Code: "GRD", Alpha2Code: "GD", Country: "Grenada"},
	"GP": {Alpha3Code: "GLP", Alpha2Code: "GP", Country: "Guadeloupe"},
	"GU": {Alpha3Code: "GUM", Alpha2Code: "GU", Country: "Guam"},
	"GT": {Alpha3Code: "GTM", Alpha2Code: "GT", Country: "Guatemala"},
	"GG": {Alpha3Code: "GGY", Alpha2Code: "GG", Country: "Guernsey"},
	"GN": {Alpha3Code: "GIN", Alpha2Code: "GN", Country: "Guinea"},
	"GW": {Alpha3Code: "GNB", Alpha2Code: "GW", Country: "Guinea-Bissau"},
	"GY": {Alpha3Code: "GUY", Alpha2Code: "GY", Country: "Guyana"},
	"HT": {Alpha3Code: "HTI", Alpha2Code: "HT", Country: "Haiti"},
	"HM": {Alpha3Code: "HMD", Alpha2Code: "HM", Country: "Heard Island and McDonald Islands"},
	"VA": {Alpha3Code: "VAT", Alpha2Code: "VA", Country: "Holy See (the)"},
	"HN": {Alpha3Code: "HND", Alpha2Code: "HN", Country: "Honduras"},
	"HK": {Alpha3Code: "HKG", Alpha2Code: "HK", Country: "Hong Kong"},
	"HU": {Alpha3Code: "HUN", Alpha2Code: "HU", Country: "Hungary"},
	"IS": {Alpha3Code: "ISL", Alpha2Code: "IS", Country: "Iceland"},
	"IN": {Alpha3Code: "IND", Alpha2Code: "IN", Country: "India"},
	"ID": {Alpha3Code: "IDN", Alpha2Code: "ID", Country: "Indonesia"},
	"IR": {Alpha3Code: "IRN", Alpha2Code: "IR", Country: "Iran (Islamic Republic of)"},
	"IQ": {Alpha3Code: "IRQ", Alpha2Code: "IQ", Country: "Iraq"},
	"IE": {Alpha3Code: "IRL", Alpha2Code: "IE", Country: "Ireland"},
	"IM": {Alpha3Code: "IMN", Alpha2Code: "IM", Country: "Isle of Man"},
	"IL": {Alpha3Code: "ISR", Alpha2Code: "IL", Country: "Israel"},
	"IT": {Alpha3Code: "ITA", Alpha2Code: "IT", Country: "Italy"},
	"JM": {Alpha3Code: "JAM", Alpha2Code: "JM", Country: "Jamaica"},
	"JP": {Alpha3Code: "JPN", Alpha2Code: "JP", Country: "Japan"},
	"JE": {Alpha3Code: "JEY", Alpha2Code: "JE", Country: "Jersey"},
	"JO": {Alpha3Code: "JOR", Alpha2Code: "JO", Country: "Jordan"},
	"KZ": {Alpha3Code: "KAZ", Alpha2Code: "KZ", Country: "Kazakhstan"},
	"KE": {Alpha3Code: "KEN", Alpha2Code: "KE", Country: "Kenya"},
	"KI": {Alpha3Code: "KIR", Alpha2Code: "KI", Country: "Kiribati"},
	"KP": {Alpha3Code: "PRK", Alpha2Code: "KP", Country: "Korea (the Democratic People's Republic of)"},
	"KR": {Alpha3Code: "KOR", Alpha2Code: "KR", Country: "Korea (the Republic of)"},
	"KW": {Alpha3Code: "KWT", Alpha2Code: "KW", Country: "Kuwait"},
	"KG": {Alpha3Code: "KGZ", Alpha2Code: "KG", Country: "Kyrgyzstan"},
	"LA": {Alpha3Code: "LAO", Alpha2Code: "LA", Country: "Lao People's Democratic Republic (the)"},
	"LV": {Alpha3Code: "LVA", Alpha2Code: "LV", Country: "Latvia"},
	"LB": {Alpha3Code: "LBN", Alpha2Code: "LB", Country: "Lebanon"},
	"LS": {Alpha3Code: "LSO", Alpha2Code: "LS", Country: "Lesotho"},
	"LR": {Alpha3Code: "LBR", Alpha2Code: "LR", Country: "Liberia"},
	"LY": {Alpha3Code: "LBY", Alpha2Code: "LY", Country: "Libya"},
	"LI": {Alpha3Code: "LIE", Alpha2Code: "LI", Country: "Liechtenstein"},
	"LT": {Alpha3Code: "LTU", Alpha2Code: "LT", Country: "Lithuania"},
	"LU": {Alpha3Code: "LUX", Alpha2Code: "LU", Country: "Luxembourg"},
	"MO": {Alpha3Code: "MAC", Alpha2Code: "MO", Country: "Macao"},
	"MG": {Alpha3Code: "MDG", Alpha2Code: "MG", Country: "Madagascar"},
	"MW": {Alpha3Code: "MWI", Alpha2Code: "MW", Country: "Malawi"},
	"MY": {Alpha3Code: "MYS", Alpha2Code: "MY", Country: "Malaysia"},
	"MV": {Alpha3Code: "MDV", Alpha2Code: "MV", Country: "Maldives"},
	"ML": {Alpha3Code: "MLI", Alpha2Code: "ML", Country: "Mali"},
	"MT": {Alpha3Code: "MLT", Alpha2Code: "MT", Country: "Malta"},
	"MH": {Alpha3Code: "MHL", Alpha2Code: "MH", Country: "Marshall Islands (the)"},
	"MQ": {Alpha3Code: "MTQ", Alpha2Code: "MQ", Country: "Martinique"},
	"MR": {Alpha3Code: "MRT", Alpha2Code: "MR", Country: "Mauritania"},
	"MU": {Alpha3Code: "MUS", Alpha2Code: "MU", Country: "Mauritius"},
	"YT": {Alpha3Code: "MYT", Alpha2Code: "YT", Country: "Mayotte"},
	"MX": {Alpha3Code: "MEX", Alpha2Code: "MX", Country: "Mexico"},
	"FM": {Alpha3Code: "FSM", Alpha2Code: "FM", Country: "Micronesia (Federated States of)"},
	"MD": {Alpha3Code: "MDA", Alpha2Code: "MD", Country: "Moldova (the Republic of)"},
	"MC": {Alpha3Code: "MCO", Alpha2Code: "MC", Country: "Monaco"},
	"MN": {Alpha3Code: "MNG", Alpha2Code: "MN", Country: "Mongolia"},
	"ME": {Alpha3Code: "MNE", Alpha2Code: "ME", Country: "Montenegro"},
	"MS": {Alpha3Code: "MSR", Alpha2Code: "MS", Country: "Montserrat"},
	"MA": {Alpha3Code: "MAR", Alpha2Code: "MA", Country: "Morocco"},
	"MZ": {Alpha3Code: "MOZ", Alpha2Code: "MZ", Country: "Mozambique"},
	"MM": {Alpha3Code: "MMR", Alpha2Code: "MM", Country: "Myanmar"},
	"NA": {Alpha3Code: "NAM", Alpha2Code: "NA", Country: "Namibia"},
	"NR": {Alpha3Code: "NRU", Alpha2Code: "NR", Country: "Nauru"},
	"NP": {Alpha3Code: "NPL", Alpha2Code: "NP", Country: "Nepal"},
	"NL": {Alpha3Code: "NLD", Alpha2Code: "NL", Country: "Netherlands (the)"},
	"NC": {Alpha3Code: "NCL", Alpha2Code: "NC", Country: "New Caledonia"},
	"NZ": {Alpha3Code: "NZL", Alpha2Code: "NZ", Country: "New Zealand"},
	"NI": {Alpha3Code: "NIC", Alpha2Code: "NI", Country: "Nicaragua"},
	"NE": {Alpha3Code: "NER", Alpha2Code: "NE", Country: "Niger (the)"},
	"NG": {Alpha3Code: "NGA", Alpha2Code: "NG", Country: "Nigeria"},
	"NU": {Alpha3Code: "NIU", Alpha2Code: "NU", Country: "Niue"},
	"NF": {Alpha3Code: "NFK", Alpha2Code: "NF", Country: "Norfolk Island"},
	"MP": {Alpha3Code: "MNP", Alpha2Code: "MP", Country: "Northern Mariana Islands (the)"},
	"NO": {Alpha3Code: "NOR", Alpha2Code: "NO", Country: "Norway"},
	"OM": {Alpha3Code: "OMN", Alpha2Code: "OM", Country: "Oman"},
	"PK": {Alpha3Code: "PAK", Alpha2Code: "PK", Country: "Pakistan"},
	"PW": {Alpha3Code: "PLW", Alpha2Code: "PW", Country: "Palau"},
	"PS": {Alpha3Code: "PSE", Alpha2Code: "PS", Country: "Palestine, State of"},
	"PA": {Alpha3Code: "PAN", Alpha2Code: "PA", Country: "Panama"},
	"PG": {Alpha3Code: "PNG", Alpha2Code: "PG", Country: "Papua New Guinea"},
	"PY": {Alpha3Code: "PRY", Alpha2Code: "PY", Country: "Paraguay"},
	"PE": {Alpha3Code: "PER", Alpha2Code: "PE", Country: "Peru"},
	"PH": {Alpha3Code: "PHL", Alpha2Code: "PH", Country: "Philippines (the)"},
	"PN": {Alpha3Code: "PCN", Alpha2Code: "PN", Country: "Pitcairn"},
	"PL": {Alpha3Code: "POL", Alpha2Code: "PL", Country: "Poland"},
	"PT": {Alpha3Code: "PRT", Alpha2Code: "PT", Country: "Portugal"},
	"PR": {Alpha3Code: "PRI", Alpha2Code: "PR", Country: "Puerto Rico"},
	"QA": {Alpha3Code: "QAT", Alpha2Code: "QA", Country: "Qatar"},
	"MK": {Alpha3Code: "MKD", Alpha2Code: "MK", Country: "Republic of North Macedonia"},
	"RO": {Alpha3Code: "ROU", Alpha2Code: "RO", Country: "Romania"},
	"RU": {Alpha3Code: "RUS", Alpha2Code: "RU", Country: "Russian Federation (the)"},
	"RW": {Alpha3Code: "RWA", Alpha2Code: "RW", Country: "Rwanda"},
	"RE": {Alpha3Code: "REU", Alpha2Code: "RE", Country: "Réunion"},
	"BL": {Alpha3Code: "BLM", Alpha2Code: "BL", Country: "Saint Barthélemy"},
	"SH": {Alpha3Code: "SHN", Alpha2Code: "SH", Country: "Saint Helena, Ascension and Tristan da Cunha"},
	"KN": {Alpha3Code: "KNA", Alpha2Code: "KN", Country: "Saint Kitts and Nevis"},
	"LC": {Alpha3Code: "LCA", Alpha2Code: "LC", Country: "Saint Lucia"},
	"MF": {Alpha3Code: "MAF", Alpha2Code: "MF", Country: "Saint Martin (French part)"},
	"PM": {Alpha3Code: "SPM", Alpha2Code: "PM", Country: "Saint Pierre and Miquelon"},
	"VC": {Alpha3Code: "VCT", Alpha2Code: "VC", Country: "Saint Vincent and the Grenadines"},
	"WS": {Alpha3Code: "WSM", Alpha2Code: "WS", Country: "Samoa"},
	"SM": {Alpha3Code: "SMR", Alpha2Code: "SM", Country: "San Marino"},
	"ST": {Alpha3Code: "STP", Alpha2Code: "ST", Country: "Sao Tome and Principe"},
	"SA": {Alpha3Code: "SAU", Alpha2Code: "SA", Country: "Saudi Arabia"},
	"SN": {Alpha3Code: "SEN", Alpha2Code: "SN", Country: "Senegal"},
	"RS": {Alpha3Code: "SRB", Alpha2Code: "RS", Country: "Serbia"},
	"SC": {Alpha3Code: "SYC", Alpha2Code: "SC", Country: "Seychelles"},
	"SL": {Alpha3Code: "SLE", Alpha2Code: "SL", Country: "Sierra Leone"},
	"SG": {Alpha3Code: "SGP", Alpha2Code: "SG", Country: "Singapore"},
	"SX": {Alpha3Code: "SXM", Alpha2Code: "SX", Country: "Sint Maarten (Dutch part)"},
	"SK": {Alpha3Code: "SVK", Alpha2Code: "SK", Country: "Slovakia"},
	"SI": {Alpha3Code: "SVN", Alpha2Code: "SI", Country: "Slovenia"},
	"SB": {Alpha3Code: "SLB", Alpha2Code: "SB", Country: "Solomon Islands"},
	"SO": {Alpha3Code: "SOM", Alpha2Code: "SO", Country: "Somalia"},
	"ZA": {Alpha3Code: "ZAF", Alpha2Code: "ZA", Country: "South Africa"},
	"GS": {Alpha3Code: "SGS", Alpha2Code: "GS", Country: "South Georgia and the South Sandwich Islands"},
	"SS": {Alpha3Code: "SSD", Alpha2Code: "SS", Country: "South Sudan"},
	"ES": {Alpha3Code: "ESP", Alpha2Code: "ES", Country: "Spain"},
	"LK": {Alpha3Code: "LKA", Alpha2Code: "LK", Country: "Sri Lanka"},
	"SD": {Alpha3Code: "SDN", Alpha2Code: "SD", Country: "Sudan (the)"},
	"SR": {Alpha3Code: "SUR", Alpha2Code: "SR", Country: "Suriname"},
	"SJ": {Alpha3Code: "SJM", Alpha2Code: "SJ", Country: "Svalbard and Jan Mayen"},
	"SE": {Alpha3Code: "SWE", Alpha2Code: "SE", Country: "Sweden"},
	"CH": {Alpha3Code: "CHE", Alpha2Code: "CH", Country: "Switzerland"},
	"SY": {Alpha3Code: "SYR", Alpha2Code: "SY", Country: "Syrian Arab Republic"},
	"TW": {Alpha3Code: "TWN", Alpha2Code: "TW", Country: "Taiwan (Province of China)"},
	"TJ": {Alpha3Code: "TJK", Alpha2Code: "TJ", Country: "Tajikistan"},
	"TZ": {Alpha3Code: "TZA", Alpha2Code: "TZ", Country: "Tanzania, United Republic of"},
	"TH": {Alpha3Code: "THA", Alpha2Code: "TH", Country: "Thailand"},
	"TL": {Alpha3Code: "TLS", Alpha2Code: "TL", Country: "Timor-Leste"},
	"TG": {Alpha3Code: "TGO", Alpha2Code: "TG", Country: "Togo"},
	"TK": {Alpha3Code: "TKL", Alpha2Code: "TK", Country: "Tokelau"},
	"TO": {Alpha3Code: "TON", Alpha2Code: "TO", Country: "Tonga"},
	"TT": {Alpha3Code: "TTO", Alpha2Code: "TT", Country: "Trinidad and Tobago"},
	"TN": {Alpha3Code: "TUN", Alpha2Code: "TN", Country: "Tunisia"},
	"TR": {Alpha3Code: "TUR", Alpha2Code: "TR", Country: "Turkey"},
	"TM": {Alpha3Code: "TKM", Alpha2Code: "TM", Country: "Turkmenistan"},
	"TC": {Alpha3Code: "TCA", Alpha2Code: "TC", Country: "Turks and Caicos Islands (the)"},
	"TV": {Alpha3Code: "TUV", Alpha2Code: "TV", Country: "Tuvalu"},
	"UG": {Alpha3Code: "UGA", Alpha2Code: "UG", Country: "Uganda"},
	"UA": {Alpha3Code: "UKR", Alpha2Code: "UA", Country: "Ukraine"},
	"AE": {Alpha3Code: "ARE", Alpha2Code: "AE", Country: "United Arab Emirates (the)"},
	"GB": {Alpha3Code: "GBR", Alpha2Code: "GB", Country: "United Kingdom of Great Britain and Northern Ireland (the)"},
	"UM": {Alpha3Code: "UMI", Alpha2Code: "UM", Country: "United States Minor Outlying Islands (the)"},
	"US": {Alpha3Code: "USA", Alpha2Code: "US", Country: "United States of America (the)"},
	"UY": {Alpha3Code: "URY", Alpha2Code: "UY", Country: "Uruguay"},
	"UZ": {Alpha3Code: "UZB", Alpha2Code: "UZ", Country: "Uzbekistan"},
	"VU": {Alpha3Code: "VUT", Alpha2Code: "VU", Country: "Vanuatu"},
	"VE": {Alpha3Code: "VEN", Alpha2Code: "VE", Country: "Venezuela (Bolivarian Republic of)"},
	"VN": {Alpha3Code: "VNM", Alpha2Code: "VN", Country: "Viet Nam"},
	"VG": {Alpha3Code: "VGB", Alpha2Code: "VG", Country: "Virgin Islands (British)"},
	"VI": {Alpha3Code: "VIR", Alpha2Code: "VI", Country: "Virgin Islands (U.S.)"},
	"WF": {Alpha3Code: "WLF", Alpha2Code: "WF", Country: "Wallis and Futuna"},
	"EH": {Alpha3Code: "ESH", Alpha2Code: "EH", Country: "Western Sahara"},
	"YE": {Alpha3Code: "YEM", Alpha2Code: "YE", Country: "Yemen"},
	"ZM": {Alpha3Code: "ZMB", Alpha2Code: "ZM", Country: "Zambia"},
	"ZW": {Alpha3Code: "ZWE", Alpha2Code: "ZW", Country: "Zimbabwe"},
	"AX": {Alpha3Code: "ALA", Alpha2Code: "AX", Country: "Åland Islands"},
}

func GetGeo(location string, baseUrl string) (lat string, lon string, address string, err error) {
    return GetGeoService().GetGeo(location, baseUrl)
}

func (s *RealGeoService) GetGeo(location string, baseUrl string) (lat string, lon string, address string, err error) {
		// TODO: this needs to be parameterized!
		if baseUrl == "" {
			return "", "", "", fmt.Errorf("base URL is empty")
		}
		htmlString, err := GetHTMLFromURL( baseUrl + "/map-embed?address=" + location, 500, false)

		if err != nil {
			return "", "", "", err
		}

		// this regex specifically captures the pattern of a lat/lon pair e.g. [40.7128, -74.0060]
		re := regexp.MustCompile(`\[\-?\+?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*\-?\+?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)\]`)
		latLon := re.FindString(htmlString)

		if latLon == "" {
			return "", "", "", fmt.Errorf("location is not valid")
		}

		latLonArr := strings.Split(latLon, ",")
		lat = latLonArr[0]
		lon = latLonArr[1]
		re = regexp.MustCompile(`[^\d.]`)
		lat = re.ReplaceAllString(lat, "")
		lon = re.ReplaceAllString(lon, "")

		// Regular expression pattern
		pattern := `"([^"]*)"\s*,\s*\` + latLon
		re = regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(htmlString)
		if len(matches) > 0 {
			address = matches[1]
		} else {
			address = "No address found"
		}

		return lat, lon, address, nil
}
