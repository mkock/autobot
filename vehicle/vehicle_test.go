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
	actual = RegCountryToString(RegCountry(0))
	expected = "DK"
	if actual != expected {
		t.Fatalf("Expected %v but got %v", expected, actual)
	}
	actual = RegCountryToString(RegCountry(1))
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
