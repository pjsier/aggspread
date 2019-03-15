package main

import (
	"math"
	"math/rand"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

type Spreader struct {
	Feature        *geojson.Feature
	AggregateValue float64
	SpreadFeatures []*geojson.Feature
}

// Return a slice of points distributed throughout spread features
func (s *Spreader) Spread() []orb.Point {
	var spreadPoints []orb.Point
	spreadTotal := s.TotalSpreadValue()
	lenSpreadFeatures := len(s.SpreadFeatures)
	totalNumPoints := int(math.Floor(s.AggregateValue))
	// Sort features in descending order by area
	sort.Slice(s.SpreadFeatures, func(i, j int) bool {
		return planar.Area(s.SpreadFeatures[i].Geometry) > planar.Area(s.SpreadFeatures[j].Geometry)
	})

	if lenSpreadFeatures == 0 {
		return spreadPoints
	}

	for _, spreadFeat := range s.SpreadFeatures {
		// Break out of the loop if already at the total number of points
		if len(spreadPoints) >= totalNumPoints {
			break
		}
		var numPoints int64
		value := planar.Area(spreadFeat.Geometry)
		portion := value / spreadTotal
		magnitude := portion * s.AggregateValue
		remainder := magnitude - math.Floor(magnitude)

		// Round magnitude to an integer semi-randomly so that if 5 spread features have
		// a magnitude of 0.2, one will have 1 point and the others will have 0
		if remainder < rand.Float64() {
			numPoints = int64(math.Floor(magnitude))
		} else {
			numPoints = int64(math.Ceil(magnitude))
		}

		var i int64
		for ; i < numPoints; i++ {
			if len(spreadPoints) >= totalNumPoints {
				break
			}
			spreadPoints = append(spreadPoints, getRandomPointInGeom(spreadFeat.Geometry))
		}
	}

	// Randomly distribute remaining points throughout spread features, starting with largest
	// TODO: Come up with a better method that doesn't sacrifice speed
	pointsToAdd := math.Floor(s.AggregateValue - float64(len(spreadPoints)))
	for i := 0; float64(i) < pointsToAdd; i++ {
		index := int(math.Floor(rand.Float64() * float64(lenSpreadFeatures)))
		spreadPoints = append(spreadPoints, getRandomPointInGeom(s.SpreadFeatures[index].Geometry))
	}

	return spreadPoints
}

// Get the sum of all feature areas to know what to distribute
// TODO: Allow spreading by prop
func (s *Spreader) TotalSpreadValue() float64 {
	var spreadValue float64
	for _, feat := range s.SpreadFeatures {
		spreadValue += planar.Area(feat.Geometry)
	}
	return spreadValue
}

func getRandomPointInGeom(geom orb.Geometry) orb.Point {
	for i := 0; i < 1000; i++ {
		bounds := geom.Bound()
		lon := bounds.Min[0] + rand.Float64()*(bounds.Max[0]-bounds.Min[0])
		lat := bounds.Min[1] + rand.Float64()*(bounds.Max[1]-bounds.Min[1])
		point := orb.Point{lon, lat}

		switch g := geom.(type) {
		case orb.Polygon:
			if planar.PolygonContains(g, point) {
				return point
			}
		case orb.MultiPolygon:
			if planar.MultiPolygonContains(g, point) {
				return point
			}
		}
	}
	return orb.Point{}
}
