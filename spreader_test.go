package main

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

func TestSpread(t *testing.T) {
	feat := geojson.NewFeature(orb.Polygon{{{0, 0}, {15, 15}, {15, 0}, {0, 0}}})
	rawJSON := `
		{
			"type": "FeatureCollection",
			"features": [
				{"type": "Feature", "geometry": {"type": "Polygon", "coordinates": [[[0, 0], [5, 5], [5, 0], [0, 0]]]}},
				{"type": "Feature", "geometry": {"type": "Polygon", "coordinates": [[[5, 5], [15, 15], [15, 5], [5, 5]]]}}
			]
		}`
	fc, err := geojson.UnmarshalFeatureCollection([]byte(rawJSON))
	if err != nil {
		panic(err)
	}
	spreader := Spreader{feat, 4.5, fc.Features}
	points := spreader.Spread()
	if planar.Area(spreader.SpreadFeatures[0].Geometry) != 50.0 {
		t.Errorf("Features should be sorted in descending order by area")
	}
	if len(points) != 4 {
		t.Errorf("Number of points returned is %d, should be 4", len(points))
	}
}

func TestTotalSpreadValue(t *testing.T) {
	feat := geojson.NewFeature(orb.Polygon{{{0, 0}, {15, 15}, {15, 0}, {0, 0}}})
	rawJSON := `
		{
			"type": "FeatureCollection",
			"features": [
				{"type": "Feature", "geometry": {"type": "Polygon", "coordinates": [[[0, 0], [5, 5], [5, 0], [0, 0]]]}},
				{"type": "Feature", "geometry": {"type": "Polygon", "coordinates": [[[5, 5], [15, 15], [15, 5], [5, 5]]]}}
			]
		}`
	fc, err := geojson.UnmarshalFeatureCollection([]byte(rawJSON))
	if err != nil {
		panic(err)
	}
	spreader := Spreader{feat, 4.5, fc.Features}
	totalValue := spreader.TotalSpreadValue()
	if totalValue != 62.5 {
		t.Errorf("Total area value should be 62.5, is %f", totalValue)
	}
}

func TestGetRandomPointInGeom(t *testing.T) {
	poly := orb.Polygon{{{0, 0}, {10, 10}, {10, 0}, {0, 0}}}
	multiPoly := orb.MultiPolygon{
		{{{15, 15}, {25, 25}, {25, 15}, {15, 15}}},
		{{{0, 0}, {10, 10}, {10, 0}, {0, 0}}},
	}
	polyPoint := getRandomPointInGeom(poly)
	multiPolyPoint := getRandomPointInGeom(multiPoly)
	if !planar.PolygonContains(poly, polyPoint) {
		t.Errorf("Polygon should contain random point")
	}
	if !planar.MultiPolygonContains(multiPoly, multiPolyPoint) {
		t.Errorf("MultiPolygon should contain random point")
	}
}
