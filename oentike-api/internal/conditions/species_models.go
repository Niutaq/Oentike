package conditions

// Coarse Open-Meteo envelopes for pilot atlas species.
// These are provisional habitat heuristics for the UI — not field surveys.
// Confidence stays low; unknown slugs must not borrow another species' score.

type speciesModel struct {
	Algorithm string
	Precip    [][2]float64
	SoilTemp  [][2]float64
	SoilMoist [][2]float64
	Wp, Wt, Wm float64
}

// Shared envelope families keep related taxa coherent without cloning one score.
// precipMM / soilTempC / soilMoist remain aliases for the original boletus v1/v2 tests.
var (
	envBoletePrecip = [][2]float64{{0, 0}, {10, 0.3}, {25, 1}, {50, 0.75}, {90, 0.2}, {120, 0.05}}
	envBoleteTemp   = [][2]float64{{5, 0.1}, {10, 0.7}, {14, 1}, {18, 0.7}, {22, 0.2}, {26, 0.05}}
	envBoleteMoist  = [][2]float64{{0.05, 0.15}, {0.15, 0.7}, {0.25, 1}, {0.35, 0.7}, {0.5, 0.2}}

	precipMM  = envBoletePrecip
	soilTempC = envBoleteTemp
	soilMoist = envBoleteMoist

	envChanterellePrecip = [][2]float64{{0, 0}, {15, 0.25}, {30, 1}, {55, 0.8}, {90, 0.25}, {130, 0.05}}
	envChanterelleTemp   = [][2]float64{{8, 0.15}, {12, 0.7}, {16, 1}, {20, 0.65}, {24, 0.2}, {28, 0.05}}
	envChanterelleMoist  = [][2]float64{{0.08, 0.15}, {0.18, 0.7}, {0.28, 1}, {0.38, 0.65}, {0.52, 0.15}}

	envSuillusPrecip = [][2]float64{{0, 0}, {8, 0.35}, {20, 1}, {45, 0.7}, {80, 0.25}, {110, 0.05}}
	envSuillusTemp   = [][2]float64{{4, 0.2}, {9, 0.75}, {13, 1}, {17, 0.7}, {21, 0.25}, {25, 0.05}}
	envSuillusMoist  = [][2]float64{{0.06, 0.2}, {0.14, 0.75}, {0.24, 1}, {0.34, 0.7}, {0.48, 0.2}}

	envAgaricPrecip = [][2]float64{{0, 0}, {5, 0.2}, {18, 0.85}, {35, 1}, {70, 0.45}, {100, 0.1}}
	envAgaricTemp   = [][2]float64{{8, 0.1}, {12, 0.55}, {16, 1}, {22, 0.85}, {26, 0.35}, {30, 0.05}}
	envAgaricMoist  = [][2]float64{{0.04, 0.2}, {0.12, 0.65}, {0.22, 1}, {0.32, 0.7}, {0.45, 0.2}}

	envPolyporePrecip = [][2]float64{{0, 0.1}, {10, 0.4}, {25, 0.85}, {50, 1}, {90, 0.55}, {140, 0.15}}
	envPolyporeTemp   = [][2]float64{{6, 0.15}, {12, 0.55}, {18, 1}, {24, 0.85}, {28, 0.4}, {32, 0.1}}
	envPolyporeMoist  = [][2]float64{{0.05, 0.25}, {0.12, 0.6}, {0.22, 0.9}, {0.35, 1}, {0.5, 0.45}}

	envAmanitaPrecip = [][2]float64{{0, 0}, {12, 0.3}, {28, 1}, {55, 0.75}, {95, 0.25}, {130, 0.05}}
	envAmanitaTemp   = [][2]float64{{6, 0.15}, {11, 0.65}, {15, 1}, {19, 0.7}, {23, 0.25}, {27, 0.05}}
	envAmanitaMoist  = [][2]float64{{0.07, 0.2}, {0.16, 0.7}, {0.26, 1}, {0.36, 0.7}, {0.5, 0.2}}
)

var speciesModels = map[string]speciesModel{
	"boletus-edulis": {
		Algorithm: "oentike-conditions/0.1.0-boletus",
		Precip:    envBoletePrecip,
		SoilTemp:  envBoleteTemp,
		SoilMoist: envBoleteMoist,
		Wp:        0.40, Wt: 0.35, Wm: 0.25,
	},
	"boletus-reticulatus": {
		Algorithm: "oentike-conditions/0.1.0-boletus-reticulatus",
		Precip:    envBoletePrecip,
		SoilTemp:  [][2]float64{{6, 0.15}, {11, 0.75}, {15, 1}, {19, 0.7}, {23, 0.25}, {27, 0.05}},
		SoilMoist: envBoleteMoist,
		Wp:        0.38, Wt: 0.37, Wm: 0.25,
	},
	"imleria-badia": {
		Algorithm: "oentike-conditions/0.1.0-imleria-badia",
		Precip:    [][2]float64{{0, 0}, {8, 0.35}, {22, 1}, {48, 0.75}, {85, 0.25}, {120, 0.05}},
		SoilTemp:  [][2]float64{{4, 0.2}, {9, 0.75}, {13, 1}, {17, 0.7}, {21, 0.25}, {25, 0.05}},
		SoilMoist: envBoleteMoist,
		Wp:        0.35, Wt: 0.35, Wm: 0.30,
	},
	"tylopilus-felleus": {
		Algorithm: "oentike-conditions/0.1.0-tylopilus-felleus",
		Precip:    envBoletePrecip,
		SoilTemp:  envBoleteTemp,
		SoilMoist: [][2]float64{{0.06, 0.2}, {0.16, 0.75}, {0.26, 1}, {0.36, 0.65}, {0.5, 0.15}},
		Wp:        0.40, Wt: 0.30, Wm: 0.30,
	},
	"suillus-luteus": {
		Algorithm: "oentike-conditions/0.1.0-suillus-luteus",
		Precip:    envSuillusPrecip,
		SoilTemp:  envSuillusTemp,
		SoilMoist: envSuillusMoist,
		Wp:        0.35, Wt: 0.30, Wm: 0.35,
	},
	"suillus-granulatus": {
		Algorithm: "oentike-conditions/0.1.0-suillus-granulatus",
		Precip:    envSuillusPrecip,
		SoilTemp:  [][2]float64{{5, 0.2}, {10, 0.75}, {14, 1}, {18, 0.7}, {22, 0.25}, {26, 0.05}},
		SoilMoist: envSuillusMoist,
		Wp:        0.35, Wt: 0.32, Wm: 0.33,
	},
	"cantharellus-cibarius": {
		Algorithm: "oentike-conditions/0.1.0-cantharellus-cibarius",
		Precip:    envChanterellePrecip,
		SoilTemp:  envChanterelleTemp,
		SoilMoist: envChanterelleMoist,
		Wp:        0.35, Wt: 0.30, Wm: 0.35,
	},
	"hygrophoropsis-aurantiaca": {
		Algorithm: "oentike-conditions/0.1.0-hygrophoropsis-aurantiaca",
		Precip:    [][2]float64{{0, 0.15}, {8, 0.45}, {20, 1}, {40, 0.7}, {75, 0.3}, {110, 0.05}},
		SoilTemp:  [][2]float64{{3, 0.25}, {8, 0.75}, {12, 1}, {16, 0.7}, {20, 0.3}, {24, 0.05}},
		SoilMoist: [][2]float64{{0.04, 0.25}, {0.12, 0.7}, {0.2, 1}, {0.3, 0.65}, {0.45, 0.2}},
		Wp:        0.30, Wt: 0.35, Wm: 0.35,
	},
	"macrolepiota-procera": {
		Algorithm: "oentike-conditions/0.1.0-macrolepiota-procera",
		Precip:    envAgaricPrecip,
		SoilTemp:  envAgaricTemp,
		SoilMoist: envAgaricMoist,
		Wp:        0.30, Wt: 0.40, Wm: 0.30,
	},
	"chlorophyllum-rhacodes": {
		Algorithm: "oentike-conditions/0.1.0-chlorophyllum-rhacodes",
		Precip:    envAgaricPrecip,
		SoilTemp:  [][2]float64{{7, 0.15}, {11, 0.6}, {15, 1}, {21, 0.85}, {25, 0.4}, {29, 0.1}},
		SoilMoist: envAgaricMoist,
		Wp:        0.30, Wt: 0.40, Wm: 0.30,
	},
	"laetiporus-sulphureus": {
		Algorithm: "oentike-conditions/0.1.0-laetiporus-sulphureus",
		Precip:    envPolyporePrecip,
		SoilTemp:  envPolyporeTemp,
		SoilMoist: envPolyporeMoist,
		Wp:        0.25, Wt: 0.45, Wm: 0.30,
	},
	"amanita-phalloides": {
		Algorithm: "oentike-conditions/0.1.0-amanita-phalloides",
		Precip:    envAmanitaPrecip,
		SoilTemp:  [][2]float64{{8, 0.15}, {12, 0.65}, {16, 1}, {20, 0.75}, {24, 0.3}, {28, 0.05}},
		SoilMoist: envAmanitaMoist,
		Wp:        0.35, Wt: 0.35, Wm: 0.30,
	},
	"amanita-muscaria": {
		Algorithm: "oentike-conditions/0.1.0-amanita-muscaria",
		Precip:    envAmanitaPrecip,
		SoilTemp:  [][2]float64{{4, 0.2}, {9, 0.7}, {13, 1}, {17, 0.7}, {21, 0.25}, {25, 0.05}},
		SoilMoist: envAmanitaMoist,
		Wp:        0.35, Wt: 0.30, Wm: 0.35,
	},
}

func hasSpeciesModel(slug string) bool {
	_, ok := speciesModels[slug]
	return ok
}

func algorithmForSpecies(slug string) string {
	if model, ok := speciesModels[slug]; ok {
		return model.Algorithm
	}
	return algorithmVersion
}
