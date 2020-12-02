package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/quadtree"
	"github.com/pjsier/aggspread/pkg/geom"
	"github.com/pjsier/aggspread/pkg/spreader"
)

func main() {
	aggPtr := flag.String("agg", "-", "File including aggregated info or '-' to read from stdin")
	propPtr := flag.String("prop", "", "Aggregated property to spreads")
	spreadPtr := flag.String("spread", "", "File to spread property throughout or '-' to read from stdin (default: value in 'agg')")
	outputPtr := flag.String("output", "-", "Optional filename to write output or '-' to write to stdout")

	flag.Parse()

	// Read GeoJSON files with aggregated properties and features to spread through
	aggFc, err := geom.LoadGeoJSONFile(*aggPtr)
	if err != nil {
		log.Fatalf("An error occurred loading aggregate GeoJSON: %s", err)
	}

	var spreadFc *geojson.FeatureCollection
	if *spreadPtr != "" {
		spreadFc, err = geom.LoadGeoJSONFile(*spreadPtr)
		if err != nil {
			log.Fatalf("An error occurred loading GeoJSON data to spread: %s", err)
		}
	} else {
		// Spread points throughout the input geometry if spread input not supplied
		spreadFc = aggFc
	}

	var qt *quadtree.Quadtree
	// Only create a quadtree if using a different set of spread features
	if *spreadPtr == "" {
		// Create a Quadtree to speed up geometry searches of spread features
		spreadFcBound := geom.FeatureCollectionBound(spreadFc)
		qt = quadtree.New(spreadFcBound)
		for _, feat := range spreadFc.Features {
			_ = qt.Add(geom.CentroidPoint{Feature: feat})
		}
	}

	var writer io.Writer
	if *outputPtr == "-" {
		writer = os.Stdout
	} else {
		writer, err = os.Create(*outputPtr)
		if err != nil {
			log.Fatalf("An error occurred creating an output file: %s", err)
		}
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	err = csvWriter.Write([]string{"lon", "lat"})
	if err != nil {
		log.Fatalf("An error occurred writing to csv: %s", err)
	}

	// Get the output channel of Spreaders
	spreaders := spreader.MakeSpreaders(aggFc, *propPtr, qt)

	// Iterate through the number of spreaders (from numFeatures) and write each to a CSV
	for spreader := range spreaders {
		for _, point := range spreader.Spread() {
			lon := fmt.Sprintf("%.6f", point[0])
			lat := fmt.Sprintf("%.6f", point[1])
			_ = csvWriter.Write([]string{lon, lat})
		}
	}
}
