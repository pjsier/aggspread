package main

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/quadtree"
)

func TestFeatureCollectionBound(t *testing.T) {
	fc := geojson.NewFeatureCollection()
	features := []*geojson.Feature{
		geojson.NewFeature(orb.Polygon{{{0, 0}, {10, 10}, {10, 0}, {0, 0}}}),
		geojson.NewFeature(orb.Polygon{{{20, 20}, {30, 30}, {30, 20}, {20, 20}}}),
		geojson.NewFeature(orb.Polygon{{{30, 30}, {40, 40}, {40, 30}, {30, 30}}}),
	}
	for _, feat := range features {
		fc.Append(feat)
	}
	bound := featureCollectionBound(fc)
	if !bound.Equal(orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{40, 40}}) {
		t.Errorf("Bound does not cover all polygons")
	}
}

func TestGetIntersectingFeatures(t *testing.T) {
	polygons := []*geojson.Feature{
		geojson.NewFeature(orb.Polygon{{{5, 5}, {10, 10}, {10, 5}, {5, 5}}}),
		geojson.NewFeature(orb.Polygon{{{20, 20}, {30, 30}, {30, 20}, {20, 20}}}),
		geojson.NewFeature(orb.Polygon{{{30, 30}, {35, 35}, {35, 30}, {30, 30}}}),
		geojson.NewFeature(orb.Polygon{{{50, 50}, {60, 60}, {60, 50}, {50, 50}}}),
	}
	fc := geojson.NewFeatureCollection()
	for _, poly := range polygons {
		fc.Append(poly)
	}
	qt := quadtree.New(featureCollectionBound(fc))
	for _, feat := range fc.Features {
		qt.Add(CentroidPoint{feat})
	}

	multiPoly := geojson.NewFeature(orb.MultiPolygon{
		{{{15, 15}, {40, 40}, {40, 15}, {15, 15}}},
		{{{0, 0}, {10, 10}, {10, 0}, {0, 0}}},
	})
	multiFc := geojson.NewFeatureCollection()
	multiFc.Append(multiPoly)
	multiQt := quadtree.New(featureCollectionBound(multiFc))
	multiQt.Add(CentroidPoint{multiFc.Features[0]})

	polyIntersect := getIntersectingFeatures(qt, multiFc.Features[0])
	if len(polyIntersect) != 3 {
		t.Errorf("Should be 3 intersecting polygons, got %d", len(polyIntersect))
	}

	noIntersect := getIntersectingFeatures(multiQt, fc.Features[3])
	if len(noIntersect) != 0 {
		t.Errorf("Should be 0 intersecting polygons, got %d", len(noIntersect))
	}

	multiIntersect := getIntersectingFeatures(multiQt, fc.Features[1])
	if len(multiIntersect) != 1 {
		t.Errorf("Should be 1 intersecting polygon, got %d", len(multiIntersect))
	}
}

func TestGeometriesIntersect(t *testing.T) {
	poly := orb.Polygon{{{0, 0}, {10, 10}, {10, 0}, {0, 0}}}
	multiPoly := orb.MultiPolygon{
		{{{15, 15}, {25, 25}, {25, 15}, {15, 15}}},
		{{{9, 9}, {10, 10}, {10, 9}, {9, 9}}},
	}

	if !geometriesIntersect(poly, multiPoly) {
		t.Errorf("Polygon should intersect MultiPolygon intersect geometry")
	}
	if !geometriesIntersect(multiPoly, poly) {
		t.Errorf("MultiPolygon should intersect Polygon intersect geometry")
	}
}
