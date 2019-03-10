package main

import (
	"math"
	"math/rand"

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
func (s *Spreader) spreadAggregateValue() []orb.Point {
	var spreadPoints []orb.Point
	spreadTotal := s.totalSpreadValue()

	for _, spreadFeat := range s.SpreadFeatures {
		spreadFeatValue := planar.Area(spreadFeat.Geometry)
		spreadFeatPortion := spreadFeatValue / spreadTotal
		numSpreadPoints := int64(math.Floor(spreadFeatPortion * s.AggregateValue))

		var i int64
		for ; i < numSpreadPoints; i++ {
			spreadPoints = append(spreadPoints, getRandomPointInGeom(spreadFeat.Geometry))
		}
	}
	return spreadPoints
}

// Get the sum of all feature areas to know what to distribute
// TODO: Allow spreading by prop
func (s *Spreader) totalSpreadValue() float64 {
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
