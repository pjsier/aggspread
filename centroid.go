package main

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

// Based on example https://github.com/paulmach/orb/blob/master/geojson/example_pointer_test.go
type CentroidPoint struct {
	*geojson.Feature
}

func (cp CentroidPoint) Point() orb.Point {
	c, _ := planar.CentroidArea(cp.Feature.Geometry)
	return c
}
