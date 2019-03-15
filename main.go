package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

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
		for _, point := range spreader.Spread() {
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
	spreaderChan := make(chan Spreader, len(aggFc.Features))
	wg := sync.WaitGroup{}

	spreadFeatureBound := featureCollectionBound(spreadFc)
	qt := quadtree.New(spreadFeatureBound)
	for _, feat := range spreadFc.Features {
		qt.Add(CentroidPoint{feat})
	}

	for _, feat := range aggFc.Features {
		wg.Add(1)
		go func(qt *quadtree.Quadtree, feat *geojson.Feature, sc chan<- Spreader) {
			sc <- Spreader{feat, feat.Properties[prop].(float64), getIntersectingFeatures(qt, feat)}
			wg.Done()
		}(qt, feat, spreaderChan)
	}
	wg.Wait()
	close(spreaderChan)
	for sf := range spreaderChan {
		spreaders = append(spreaders, sf)
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

func getIntersectingFeatures(qt *quadtree.Quadtree, feat *geojson.Feature) []*geojson.Feature {
	var overlap []*geojson.Feature
	for _, featPtr := range qt.InBound(nil, feat.Geometry.Bound()) {
		overlapFeat := featPtr.(CentroidPoint).Feature
		if geometriesIntersect(feat.Geometry, overlapFeat.Geometry) || geometriesIntersect(overlapFeat.Geometry, feat.Geometry) {
			overlap = append(overlap, overlapFeat)
		}
	}
	return overlap
}

func geometriesIntersect(geom orb.Geometry, intersectGeom orb.Geometry) bool {
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
