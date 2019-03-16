package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
	"github.com/pkg/errors"
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
	spreadFeatureBound := featureCollectionBound(spreadFc)
	qt := quadtree.New(spreadFeatureBound)
	for _, feat := range spreadFc.Features {
		qt.Add(CentroidPoint{feat})
	}

	featureChan := make(chan *geojson.Feature)
	spreaderChan := make(chan Spreader)

	// Start up worker goroutines to process the data. 8 can be any number you want. You could
	// use the value of runtime.GOMAXPROCS, but I would experiment to see what gives the best performance.
	// You can use lots and lots of goroutines, but if you're CPU bound that won't make things faster and
	// will add some overhead.
	for i := 0; i < 8; i++ {
		makeSpreaders(featureChan, spreaderChan, qt, prop)
	}

	// Keep track of how many features we've seen so that we know how many Spreaders to expect later on
	// You could just get this number using len(aggFc.Features), but this design will get you closer to
	// streaming data through the program instead of loading it all into memory at once.
	numFeatures := 0
	for _, feat := range aggFc.Features {
		featureChan <- feat
		numFeatures++
	}

	// Closing this channel will stop the range loops within the makeSpreaders goroutines
	close(featureChan)

	var spreaders []Spreader
	for i := 0; i < numFeatures; i++ {
		spreaders = append(spreaders, <-spreaderChan)
	}

	return spreaders
}

// Starts a goroutine that pulls features off featureChan and pushes a resulting Spreader back onto
// spreaderChan
func makeSpreaders(featureChan <-chan *geojson.Feature, spreaderChan chan<- Spreader, qt *quadtree.Quadtree, prop string) {
	go func() {
		for feat := range featureChan {
			spreaderChan <- Spreader{feat, feat.Properties[prop].(float64), getIntersectingFeatures(qt, feat)}
		}
	}()
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
