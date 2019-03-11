package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
)

func main() {
	aggPtr := flag.String("agg", "", "File including aggregated info or '-' to read from stdin")
	propPtr := flag.String("prop", "", "Aggregated property")
	spreadPtr := flag.String("spread", "", "File to spread property throughout")
	outputPtr := flag.String("output", "", "CSV filename to write to or '-' to write to stdout")

	flag.Parse()

	// Read GeoJSON files with aggregated properties and features to spread through
	aggFeatures, err := loadGeoJSONFile(*aggPtr)
	if err != nil {
		panic(err)
	}
	spreadFeatures, err := loadGeoJSONFile(*spreadPtr)
	if err != nil {
		panic(err)
	}

	spreaders := aggFeaturesToSpread(aggFeatures, spreadFeatures, *propPtr)

	var writer io.Writer
	if *outputPtr == "-" {
		writer = os.Stdout
	} else {
		writer, err = os.Create(*outputPtr)
		if err != nil {
			panic(err)
		}
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	err = csvWriter.Write([]string{"lon", "lat"})

	for _, spreader := range spreaders {
		for _, point := range spreader.spreadAggregateValue() {
			lon := fmt.Sprintf("%.6f", point[0])
			lat := fmt.Sprintf("%.6f", point[1])
			csvWriter.Write([]string{lon, lat})
		}
	}
	if err != nil {
		panic(err)
	}
}

func loadGeoJSONFile(filename string) (*geojson.FeatureCollection, error) {
	var data []byte
	var err error
	var features = &geojson.FeatureCollection{}
	if filename == "" {
		return features, errors.New("Filename must not be blank")
	}

	if filename == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(filename)
	}

	if err != nil {
		return features, err
	}
	features, err = geojson.UnmarshalFeatureCollection(data)
	if err != nil {
		return features, err
	}
	return features, nil
}

// Takes a feature collection of aggregate features, returns a slice of Spreader objects
// that can distribute the aggregated values throughout the spread features
func aggFeaturesToSpread(aggFc *geojson.FeatureCollection, spreadFc *geojson.FeatureCollection, prop string) []Spreader {
	var spreaders []Spreader

	spreadFeatureBound := featureCollectionBound(spreadFc)
	qt := quadtree.New(spreadFeatureBound)
	for _, feat := range spreadFc.Features {
		qt.Add(CentroidPoint{feat})
	}

	for _, feat := range aggFc.Features {
		var spreadFeatures []*geojson.Feature
		for _, spreadPtr := range qt.InBound(nil, feat.Geometry.Bound()) {
			spreadFeat := spreadPtr.(CentroidPoint).Feature
			if geometriesIntersect(feat.Geometry, spreadFeat.Geometry) {
				spreadFeatures = append(spreadFeatures, spreadFeat)
			}
		}
		spreaders = append(spreaders, Spreader{feat, feat.Properties[prop].(float64), spreadFeatures})
	}
	return spreaders
}

func featureCollectionBound(fc *geojson.FeatureCollection) orb.Bound {
	var bound orb.Bound
	bound = fc.Features[0].Geometry.Bound()
	for _, feat := range fc.Features[1:] {
		bound = bound.Union(feat.Geometry.Bound())
	}
	return bound
}

func geometriesIntersect(geom orb.Geometry, intersectGeom orb.Geometry) bool {
	switch g := geom.(type) {
	case orb.Polygon:
		intersects := polygonOverlaps(g, intersectGeom)
		if intersects {
			return intersects
		}
	case orb.MultiPolygon:
		intersects := multiPolygonOverlaps(g, intersectGeom)
		if intersects {
			return intersects
		}
	}

	return false
}

func polygonOverlaps(geom orb.Polygon, intersectGeom orb.Geometry) bool {
	switch g := intersectGeom.(type) {
	case orb.Polygon:
		for _, ring := range g {
			for _, point := range ring {
				if planar.PolygonContains(geom, point) {
					return true
				}
			}
		}
	case orb.MultiPolygon:
		for _, polygon := range g {
			for _, ring := range polygon {
				for _, point := range ring {
					if planar.PolygonContains(geom, point) {
						return true
					}
				}
			}
		}
	}
	return false
}

func multiPolygonOverlaps(geom orb.MultiPolygon, intersectGeom orb.Geometry) bool {
	switch g := intersectGeom.(type) {
	case orb.Polygon:
		for _, ring := range g {
			for _, point := range ring {
				if planar.MultiPolygonContains(geom, point) {
					return true
				}
			}
		}
	case orb.MultiPolygon:
		for _, polygon := range g {
			for _, ring := range polygon {
				for _, point := range ring {
					if planar.MultiPolygonContains(geom, point) {
						return true
					}
				}
			}
		}
	}
	return false
}
