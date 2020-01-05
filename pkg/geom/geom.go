package geom

import (
	"math/rand"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
)

// CentroidPoint is used to manage generating a quadtree while referencing a GeoJSON Feature
// Based on example https://github.com/paulmach/orb/blob/master/geojson/example_pointer_test.go
type CentroidPoint struct {
	*geojson.Feature
}

// Point returns an orb.Point object from a CentroidPoint struct
func (cp CentroidPoint) Point() orb.Point {
	c, _ := planar.CentroidArea(cp.Feature.Geometry)
	return c
}

// FeatureCollectionBound returns the bounds of a FeatureCollection
func FeatureCollectionBound(fc *geojson.FeatureCollection) orb.Bound {
	bound := fc.Features[0].Geometry.Bound()
	for _, feat := range fc.Features[1:] {
		bound = bound.Union(feat.Geometry.Bound())
	}
	return bound
}

// IntersectingFeatures returns all features intersecting a given feature in a quadtree
func IntersectingFeatures(qt *quadtree.Quadtree, feat *geojson.Feature) []*geojson.Feature {
	var overlap []*geojson.Feature
	for _, featPtr := range qt.InBound(nil, feat.Geometry.Bound()) {
		overlapFeat := featPtr.(CentroidPoint).Feature
		if GeometriesIntersect(feat.Geometry, overlapFeat.Geometry) || GeometriesIntersect(overlapFeat.Geometry, feat.Geometry) {
			overlap = append(overlap, overlapFeat)
		}
	}
	return overlap
}

// GeometriesIntersect checks whether two geometries intersect each other
func GeometriesIntersect(geom orb.Geometry, intersectGeom orb.Geometry) bool {
	var polyRange []orb.Polygon

	switch g := intersectGeom.(type) {
	case orb.Polygon:
		polyRange = []orb.Polygon{g}
	case orb.MultiPolygon:
		polyRange = g
	}
	for _, polygon := range polyRange {
		for _, ring := range polygon {
			for _, point := range ring {
				switch gt := geom.(type) {
				case orb.Polygon:
					if planar.PolygonContains(gt, point) {
						return true
					}
				case orb.MultiPolygon:
					if planar.MultiPolygonContains(gt, point) {
						return true
					}
				}
			}
		}
	}
	return false
}

// RandomPointInGeom generates a random point within a given geometry
func RandomPointInGeom(geom orb.Geometry) orb.Point {
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
