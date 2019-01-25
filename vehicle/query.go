package vehicle

import "strings"

// Query contains the search- and filter options for performing a query against the store.
type Query struct {
	Limit    int64
	Type     string
	Brand    string
	Model    string
	FuelType string
}

type preparedQuery struct {
	limit       int64
	vehicleType Type
	byType      bool
	brand       string
	model       string
	fuelType    string
}

func (pq preparedQuery) validates(v Vehicle) bool {
	var checks, passed int
	if pq.byType {
		checks++
		if v.Type == pq.vehicleType {
			passed++
		}
	}
	if pq.brand != "" {
		checks++
		if strings.EqualFold(v.Brand, pq.brand) {
			passed++
		}
	}
	if pq.model != "" {
		checks++
		if strings.EqualFold(v.Model, pq.model) {
			passed++
		}
	}
	if pq.fuelType != "" {
		checks++
		if strings.EqualFold(v.FuelType, pq.fuelType) {
			passed++
		}
	}
	return passed == checks
}

func prepareQuery(q Query) preparedQuery {
	return preparedQuery{limit: q.Limit, vehicleType: TypeFromString(q.Type), byType: q.Type != "", brand: q.Brand, model: q.Model, fuelType: q.FuelType}
}
