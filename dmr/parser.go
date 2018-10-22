package dmr

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/OmniCar/autobot/autoservice"
)

// XMLParser represents an XML parser.
type XMLParser struct {
}

// NewXMLParser creates a new XML parser.
func NewXMLParser() *XMLParser {
	return &XMLParser{}
}

// ParseExcerpt parses XML file using XML decoding.
func (p *XMLParser) ParseExcerpt(id int, lines <-chan []string, parsed chan<- autoservice.Vehicle, done chan<- int) {
	var proc, keep int // How many excerpts did we process and keep?
	var stat vehicleStat
	var hash uint64
	for excerpt := range lines {
		if err := xml.Unmarshal([]byte(strings.Join(excerpt, "\n")), &stat); err != nil {
			panic(err) // We _could_ skip it, but it's better to halt execution here.
		}
		if stat.Type == 1 {
			regDate, err := time.Parse("2006-01-02", stat.Info.FirstRegDate[:10])
			if err != nil {
				fmt.Printf("Error: Unable to parse first registration date: %s\n", stat.Info.FirstRegDate)
				continue
			}
			vehicle := autoservice.Vehicle{
				MetaData:     autoservice.Meta{Source: stat.Info.Source, Ident: stat.Ident, LastUpdated: time.Now()},
				RegNr:        strings.ToUpper(stat.RegNo),
				VIN:          strings.ToUpper(stat.Info.VIN),
				Brand:        stat.Info.Designation.BrandTypeName, // @TODO Title-case brand name.
				Model:        stat.Info.Designation.Model.Name,    // @TODO Title-case model name.
				FuelType:     stat.Info.Engine.Fuel.FuelType,      // @TODO Title-case fuel-type name.
				FirstRegDate: regDate,
			}
			if hash, err = hashstructure.Hash(vehicle, nil); err != nil {
				fmt.Printf("Error: Unable to hash Vehicle struct with Ident: %d\n", vehicle.MetaData.Ident)
				continue
			}
			vehicle.MetaData.Hash = hash
			parsed <- vehicle
			keep++
		}
		proc++
	}
	fmt.Printf("XML-worker %d finished processing %d excerpts, kept %d\n", id, proc, keep)
	done <- id
}
