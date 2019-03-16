package main

import (
	"encoding/csv"
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
	"github.com/pkg/errors"
)

const WORKER_COUNT = 8

func main() {
	aggPtr := flag.String("agg", "", "File including aggregated info or '-' to read from stdin")
	propPtr := flag.String("prop", "", "Aggregated property")
	spreadPtr := flag.String("spread", "", "File to spread property throughout")
	outputPtr := flag.String("output", "", "CSV filename to write to or '-' to write to stdout")

	flag.Parse()

	// Read GeoJSON files with aggregated properties and features to spread through
	aggFc, err := loadGeoJSONFile(*aggPtr)
	if err != nil {
		panic(err)
	}
	spreadFc, err := loadGeoJSONFile(*spreadPtr)
	if err != nil {
		panic(err)
	}

	// Create a Quadtree to speed up geometry searches of spread features
	spreadFcBound := featureCollectionBound(spreadFc)
	qt := quadtree.New(spreadFcBound)
	for _, feat := range spreadFc.Features {
		qt.Add(CentroidPoint{feat})
	}

	// Get the total number of aggregated features to spread and a channel of Spreaders
	numFeatures, spreaderChan := makeSpreaders(aggFc, qt, *propPtr)

	var writer io.Writer
	if *outputPtr == "-" {
		writer = os.Stdout
	} else {
		writer, err = os.Create(*outputPtr)
		if err != nil {
			panic(err)
		}
	}

	var mu sync.Mutex
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	err = csvWriter.Write([]string{"lon", "lat"})

	// Iterate through the number of spreaders (from numFeatures) and write each to a CSV
	for i := 0; i < numFeatures; i++ {
		spreader := <-spreaderChan
		mu.Lock()
		for _, point := range spreader.Spread() {
			lon := fmt.Sprintf("%.6f", point[0])
			lat := fmt.Sprintf("%.6f", point[1])
			csvWriter.Write([]string{lon, lat})
		}
		mu.Unlock()
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
func makeSpreaders(fc *geojson.FeatureCollection, qt *quadtree.Quadtree, prop string) (int, <-chan Spreader) {
	featureChan := make(chan *geojson.Feature)
	spreaderChan := make(chan Spreader)

	// Start up worker goroutines to process the data. 8 can be any number you want. You could
	// use the value of runtime.GOMAXPROCS, but I would experiment to see what gives the best performance.
	// You can use lots and lots of goroutines, but if you're CPU bound that won't make things faster and
	// will add some overhead.
	for i := 0; i < WORKER_COUNT; i++ {
		updateSpreaderChan(featureChan, spreaderChan, qt, prop)
	}

	// Keep track of how many features we've seen so that we know how many Spreaders to expect later on
	// You could just get this number using len(fc.Features), but this design will get you closer to
	// streaming data through the program instead of loading it all into memory at once.
	numFeatures := 0
	for _, feat := range fc.Features {
		go func(feat *geojson.Feature) {
			featureChan <- feat
		}(feat)
		numFeatures++
	}

	return numFeatures, spreaderChan
}

// Starts a goroutine that pulls features off featureChan and pushes a resulting Spreader back onto
// spreaderChan
func updateSpreaderChan(featureChan <-chan *geojson.Feature, spreaderChan chan<- Spreader, qt *quadtree.Quadtree, prop string) {
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
