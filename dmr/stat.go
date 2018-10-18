package dmr

// <ns:Statistik>
type vehicleStat struct {
	Ident uint64      `xml:"KoeretoejIdent"`
	Type  uint64      `xml:"KoeretoejArtNummer"`
	RegNo string      `xml:"RegistreringNummerNummer"`
	Info  vehicleInfo `xml:"KoeretoejOplysningGrundStruktur"`
}

// <ns:KoeretoejOplysningGrundStruktur>
type vehicleInfo struct {
	Source       string             `xml:"KoeretoejOplysningOprettetUdFra"`
	Status       string             `xml:"KoeretoejOplysningStatus"`
	StatusDate   string             `xml:"KoeretoejOplysningStatusDato"`
	Variant      string             `xml:"KoeretoejVariantTypeNavn"`
	VIN          string             `xml:"KoeretoejOplysningStelNummer"`
	FirstRegDate string             `xml:"KoeretoejOplysningFoersteRegistreringDato"`
	Engine       vehicleEngine      `xml:"KoeretoejMotorStruktur"`
	Designation  vehicleDesignation `xml:"KoeretoejBetegnelseStruktur"`
}

// <ns:Model>
type vehicleModel struct {
	Type uint64 `xml:"KoeretoejModelTypeNummer"`
	Name string `xml:"KoeretoejModelTypeNavn"`
}

// <ns:Variant>
type vehicleVariant struct {
	Type uint64 `xml:"KoeretoejVariantTypeNummer"`
	Name string `xml:"KoeretoejVariantTypeNavn"`
}

// <ns:Type>
type vehicleType struct {
	Type uint64 `xml:"KoeretoejTypeTypeNummer"`
	Name string `xml:"KoeretoejTypeTypeNavn"`
}

// <ns:KoeretoejMotorStruktur>
type vehicleEngine struct {
	Fuel vehicleFuel `xml:"DrivkraftTypeStruktur"`
}

// <ns:DrivkraftTypeStruktur>
type vehicleFuel struct {
	FuelType string `xml:"DrivkraftTypeNavn"`
}

// <ns:KoeretoejBetegnelseStruktur>
type vehicleDesignation struct {
	BrandTypeNr   uint64         `xml:"KoeretoejMaerkeTypeNummer"`
	BrandTypeName string         `xml:"KoeretoejMaerkeTypeNavn"`
	Model         vehicleModel   `xml:"Model"`
	Variant       vehicleVariant `xml:"Variant"`
	Type          vehicleType    `xml:"Type"`
}
