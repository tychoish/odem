package models

type SingingLocality string

func NewSingingLocality(in string) SingingLocality { return SingingLocality(in) }

func (s SingingLocality) Valid() bool {
	switch s {
	case LocalityAlaska, LocalityAlabama, LocalityArkansas, LocalityArizona,
		LocalityCalifornia, LocalityColorado, LocalityConnecticut, LocalityDC,
		LocalityFlorida, LocalityGeorgia, LocalityIowa, LocalityIllinois,
		LocalityIndiana, LocalityKansas, LocalityKentucky, LocalityLouisiana,
		LocalityMassachusetts, LocalityMaryland, LocalityMaine, LocalityMichigan,
		LocalityMinnesota, LocalityMissouri, LocalityMississippi, LocalityNorthCarolina,
		LocalityNebraska, LocalityNewHampshire, LocalityNewJersey, LocalityNewMexico,
		LocalityNewYork, LocalityOhio, LocalityOklahoma, LocalityOregon,
		LocalityPennsylvania, LocalityRhodeIsland, LocalitySouthCarolina, LocalitySouthDakota,
		LocalityTennessee, LocalityTexas, LocalityUtah, LocalityVirginia,
		LocalityVermont, LocalityWashington, LocalityWisconsin, LocalityWestVirginia,
		LocalityBritishColumbia, LocalityOntario, LocalityQuebec,
		LocalityAustralianCapitalTerritory, LocalityNewSouthWales, LocalityVictoria,
		LocalityEngland, LocalityNorthernIreland, LocalityScotland, LocalityWales, LocalityMunster:
		return true
	}
	return false
}

func AllLocalities() []SingingLocality {
	return []SingingLocality{
		LocalityAlaska, LocalityAlabama, LocalityArkansas, LocalityArizona,
		LocalityCalifornia, LocalityColorado, LocalityConnecticut, LocalityDC,
		LocalityFlorida, LocalityGeorgia, LocalityIowa, LocalityIllinois,
		LocalityIndiana, LocalityKansas, LocalityKentucky, LocalityLouisiana,
		LocalityMassachusetts, LocalityMaryland, LocalityMaine, LocalityMichigan,
		LocalityMinnesota, LocalityMissouri, LocalityMississippi, LocalityNorthCarolina,
		LocalityNebraska, LocalityNewHampshire, LocalityNewJersey, LocalityNewMexico,
		LocalityNewYork, LocalityOhio, LocalityOklahoma, LocalityOregon,
		LocalityPennsylvania, LocalityRhodeIsland, LocalitySouthCarolina, LocalitySouthDakota,
		LocalityTennessee, LocalityTexas, LocalityUtah, LocalityVirginia,
		LocalityVermont, LocalityWashington, LocalityWisconsin, LocalityWestVirginia,
		LocalityBritishColumbia, LocalityOntario, LocalityQuebec,
		LocalityAustralianCapitalTerritory, LocalityNewSouthWales, LocalityVictoria,
		LocalityEngland, LocalityNorthernIreland, LocalityScotland, LocalityWales, LocalityMunster,
	}
}

const (
	// US states
	LocalityAlaska        SingingLocality = "AK"
	LocalityAlabama       SingingLocality = "AL"
	LocalityArkansas      SingingLocality = "AR"
	LocalityArizona       SingingLocality = "AZ"
	LocalityCalifornia    SingingLocality = "CA"
	LocalityColorado      SingingLocality = "CO"
	LocalityConnecticut   SingingLocality = "CT"
	LocalityDC            SingingLocality = "DC"
	LocalityFlorida       SingingLocality = "FL"
	LocalityGeorgia       SingingLocality = "GA"
	LocalityIowa          SingingLocality = "IA"
	LocalityIllinois      SingingLocality = "IL"
	LocalityIndiana       SingingLocality = "IN"
	LocalityKansas        SingingLocality = "KS"
	LocalityKentucky      SingingLocality = "KY"
	LocalityLouisiana     SingingLocality = "LA"
	LocalityMassachusetts SingingLocality = "MA"
	LocalityMaryland      SingingLocality = "MD"
	LocalityMaine         SingingLocality = "ME"
	LocalityMichigan      SingingLocality = "MI"
	LocalityMinnesota     SingingLocality = "MN"
	LocalityMissouri      SingingLocality = "MO"
	LocalityMississippi   SingingLocality = "MS"
	LocalityNorthCarolina SingingLocality = "NC"
	LocalityNebraska      SingingLocality = "NE"
	LocalityNewHampshire  SingingLocality = "NH"
	LocalityNewJersey     SingingLocality = "NJ"
	LocalityNewMexico     SingingLocality = "NM"
	LocalityNewYork       SingingLocality = "NY"
	LocalityOhio          SingingLocality = "OH"
	LocalityOklahoma      SingingLocality = "OK"
	LocalityOregon        SingingLocality = "OR"
	LocalityPennsylvania  SingingLocality = "PA"
	LocalityRhodeIsland   SingingLocality = "RI"
	LocalitySouthCarolina SingingLocality = "SC"
	LocalitySouthDakota   SingingLocality = "SD"
	LocalityTennessee     SingingLocality = "TN"
	LocalityTexas         SingingLocality = "TX"
	LocalityUtah          SingingLocality = "UT"
	LocalityVirginia      SingingLocality = "VA"
	LocalityVermont       SingingLocality = "VT"
	LocalityWashington    SingingLocality = "WA"
	LocalityWisconsin     SingingLocality = "WI"
	LocalityWestVirginia  SingingLocality = "WV"

	// Canadian provinces
	LocalityBritishColumbia SingingLocality = "British Columbia"
	LocalityOntario         SingingLocality = "ON"
	LocalityQuebec          SingingLocality = "QC"

	// Australian states/territories
	LocalityAustralianCapitalTerritory SingingLocality = "ACT"
	LocalityNewSouthWales              SingingLocality = "NSW"
	LocalityVictoria                   SingingLocality = "VIC"

	// UK and Ireland
	LocalityEngland         SingingLocality = "England"
	LocalityNorthernIreland SingingLocality = "Northern Ireland"
	LocalityScotland        SingingLocality = "Scotland"
	LocalityWales           SingingLocality = "Wales"
	LocalityMunster         SingingLocality = "Munster"
)
