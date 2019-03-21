package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/paulmach/orb/quadtree"
	"github.com/pjsier/aggspread/pkg/geom"
	"github.com/pjsier/aggspread/pkg/spreader"
)

func main() {
	aggPtr := flag.String("agg", "", "File including aggregated info or '-' to read from stdin")
	propPtr := flag.String("prop", "", "Aggregated property")
	spreadPtr := flag.String("spread", "", "File to spread property throughout")
	outputPtr := flag.String("output", "", "CSV filename to write to or '-' to write to stdout")

	flag.Parse()

	// Read GeoJSON files with aggregated properties and features to spread through
	aggFc, err := geom.LoadGeoJSONFile(*aggPtr)
	if err != nil {
		panic(err)
	}
	spreadFc, err := geom.LoadGeoJSONFile(*spreadPtr)
	if err != nil {
		panic(err)
	}

	// Create a Quadtree to speed up geometry searches of spread features
	spreadFcBound := geom.FeatureCollectionBound(spreadFc)
	qt := quadtree.New(spreadFcBound)
	for _, feat := range spreadFc.Features {
		qt.Add(geom.CentroidPoint{feat})
	}

	// Get the total number of aggregated features to spread and a channel of Spreaders
	numFeatures, spreaderChan := spreader.MakeSpreaders(aggFc, qt, *propPtr)

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

	// Iterate through the number of spreaders (from numFeatures) and write each to a CSV
	for i := 0; i < numFeatures; i++ {
		spreader := <-spreaderChan
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
