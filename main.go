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
)

func main() {
	aggPtr := flag.String("agg", "", "File including aggregated info")
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
	var features = &geojson.FeatureCollection{}
	if filename == "" {
		return features, errors.New("Filename must not be blank")
	}

	data, err := ioutil.ReadFile(filename)
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

	for _, feat := range aggFc.Features {
		var spreadFeatures []*geojson.Feature
		for _, spreadFeat := range spreadFc.Features {
			if geometriesIntersect(feat.Geometry, spreadFeat.Geometry) {
				spreadFeatures = append(spreadFeatures, spreadFeat)
			}
		}
		spreaders = append(spreaders, Spreader{feat, feat.Properties[prop].(float64), spreadFeatures})
	}
	return spreaders
}

func geometriesIntersect(geom orb.Geometry, intersectGeom orb.Geometry) bool {
	geomBound := geom.Bound()
	intersectBound := intersectGeom.Bound()

	// Check if bounding box overlaps first
	if !geomBound.Contains(intersectBound.Min) && !geomBound.Contains(intersectBound.Max) {
		return false
	}

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
