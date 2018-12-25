package vehicle

import (
	"testing"
	"time"
)

func TestRegCountryFromString(t *testing.T) {
	actual := RegCountryFromString("DK")
	expected := RegCountry(0)
	if actual != expected {
		t.Fatalf("Expected %v but got %v", expected, actual)
	}
	actual = RegCountryFromString("NO")
	expected = RegCountry(1)
	if actual != expected {
		t.Fatalf("Expected %v but got %v", expected, actual)
	}
}

func TestRegCountryToString(t *testing.T) {
	var actual, expected string
	actual = RegCountry(0).String()
	expected = "DK"
	if actual != expected {
		t.Fatalf("Expected %v but got %v", expected, actual)
	}
	actual = RegCountry(1).String()
	expected = "NO"
	if actual != expected {
		t.Fatalf("Expected %v but got %v", expected, actual)
	}
}

func TestGenHash(t *testing.T) {
	var err error
	v := Vehicle{}
	v.MetaData = Meta{0, "A Source", DK, 0, time.Now(), false}
	if err = v.GenHash(); err != nil {
		t.Fatal(err)
	}
	expected := v.MetaData.Hash
	v.MetaData.Source = "Another Source"
	actual := v.MetaData.Hash
	if err = v.GenHash(); err != nil {
		t.Fatal(err)
	}
	if v.MetaData.Hash == 0 {
		t.Fatal("Expected a hash to be generated")
	}
	if actual != expected {
		t.Fatal("Expected hash to be unaffected by changes to metadata")
	}
}

func TestPrettyBrandName(t *testing.T) {
	cases := map[string]string{
		"bmw":     "BMW",
		"PEUGEOT": "Peugeot",
		"ds":      "DS",
		"MINI":    "Mini",
	}
	var actual string
	for in, expected := range cases {
		actual = PrettyBrandName(in)
		if actual != expected {
			t.Fatalf("Expected %v but got %v", expected, actual)
		}
	}
}

func TestPrettyFuelType(t *testing.T) {
	cases := map[string]string{
		"diesel": "Diesel",
		"poWEr":  "Power",
	}
	var actual string
	for in, expected := range cases {
		actual = PrettyFuelType(in)
		if actual != expected {
			t.Fatalf("Expected %v but got %v", expected, actual)
		}
	}
}

func TestQueryValidates(t *testing.T) {
	vehicles := map[string]Vehicle{
		"FordMondeo": Vehicle{
			Type:         Car,
			Brand:        "Ford",
			Model:        "Mondeo",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
		"ToyotaCorolla": Vehicle{
			Type:         Car,
			Brand:        "Toyota",
			Model:        "Corolla",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
		"FordFiesta": Vehicle{
			Type:         Car,
			Brand:        "Ford",
			Model:        "Fiesta",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
	}
	results := map[string]bool{
		"FordMondeo":    true,
		"ToyotaCorolla": false,
		"FordFiesta":    true,
	}
	q := Query{
		Limit: 10,
		Brand: "Ford",
	}
	pq := prepareQuery(q)
	for name, veh := range vehicles {
		valid := pq.validates(veh)
		if results[name] != valid {
			t.Fatalf("Expected %q to validate query", name)
		}
	}
}
func TestQueryValidatesMultiple(t *testing.T) {
	vehicles := map[string]Vehicle{
		"FordMondeo": Vehicle{
			Type:         Car,
			Brand:        "Ford",
			Model:        "Mondeo",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
		"ToyotaCorolla": Vehicle{
			Type:         Car,
			Brand:        "Toyota",
			Model:        "Corolla",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
		"FordFiesta#1": Vehicle{
			Type:         Car,
			Brand:        "Ford",
			Model:        "Fiesta",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
		"FordFiesta#2": Vehicle{
			Type:         Car,
			Brand:        "Ford",
			Model:        "Fiesta",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
	}
	results := map[string]bool{
		"FordMondeo":    false,
		"ToyotaCorolla": false,
		"FordFiesta#1":  true,
		"FordFiesta#2":  true,
	}
	q := Query{
		Limit: 10,
		Brand: "Ford",
		Model: "Fiesta",
	}
	pq := prepareQuery(q)
	for name, veh := range vehicles {
		valid := pq.validates(veh)
		if results[name] != valid {
			t.Fatalf("Expected validation of %q to equal %v", name, results[name])
		}
	}
}

func TestQueryValidatesEmptyCond(t *testing.T) {
	vehicles := map[string]Vehicle{
		"FordMondeo": Vehicle{
			Type:         Car,
			Brand:        "Ford",
			Model:        "Mondeo",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
		"ToyotaCorolla": Vehicle{
			Type:         Car,
			Brand:        "Toyota",
			Model:        "Corolla",
			FuelType:     "Benzin",
			FirstRegDate: time.Now(),
		},
	}
	results := map[string]bool{
		"FordMondeo":    true,
		"ToyotaCorolla": true,
	}
	q := Query{
		Limit: 10,
		Type:  "",
	}
	pq := prepareQuery(q)
	for name, veh := range vehicles {
		valid := pq.validates(veh)
		if results[name] != valid {
			t.Fatalf("Expected validation of %q to equal %v", name, results[name])
		}
	}
}
