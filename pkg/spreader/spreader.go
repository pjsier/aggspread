package spreader

import (
	"math"
	"math/rand"
	"sort"
	"sync"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
	"github.com/pjsier/aggspread/pkg/geom"
)

const workerCount = 8

// Spreader manages spreading a Feature into SpreadFeatures based on the AggregateValue
type Spreader struct {
	Feature        *geojson.Feature
	AggregateValue float64
	SpreadFeatures []*geojson.Feature
}

// Spread returns a slice of points distributed throughout spread features
func (s *Spreader) Spread() []orb.Point {
	var spreadPoints []orb.Point
	spreadTotal := s.TotalSpreadValue()
	lenSpreadFeatures := len(s.SpreadFeatures)
	totalNumPoints := int(math.Floor(s.AggregateValue))

	// Sort features in descending order by area
	sort.Slice(s.SpreadFeatures, func(i, j int) bool {
		return planar.Area(s.SpreadFeatures[i].Geometry) > planar.Area(s.SpreadFeatures[j].Geometry)
	})

	if lenSpreadFeatures == 0 {
		return spreadPoints
	}

	for _, spreadFeat := range s.SpreadFeatures {
		// Break out of the loop if already at the total number of points
		if len(spreadPoints) >= totalNumPoints {
			break
		}
		var numPoints int64
		value := planar.Area(spreadFeat.Geometry)
		portion := value / spreadTotal
		magnitude := portion * s.AggregateValue
		remainder := magnitude - math.Floor(magnitude)

		// Round magnitude to an integer semi-randomly so that if 5 spread features have
		// a magnitude of 0.2, one will have 1 point and the others will have 0
		if remainder < rand.Float64() {
			numPoints = int64(math.Floor(magnitude))
		} else {
			numPoints = int64(math.Ceil(magnitude))
		}

		for i := 0; i < int(numPoints); i++ {
			if len(spreadPoints) >= totalNumPoints {
				break
			}
			spreadPoints = append(spreadPoints, geom.RandomPointInGeom(spreadFeat.Geometry))
		}
	}

	// Randomly distribute remaining points throughout spread features
	for {
		if len(spreadPoints) >= int(math.Floor(s.AggregateValue)) {
			break
		}
		index := int(math.Floor(rand.Float64() * float64(lenSpreadFeatures)))
		spreadPoints = append(spreadPoints, geom.RandomPointInGeom(s.SpreadFeatures[index].Geometry))
	}

	return spreadPoints
}

// TotalSpreadValue returns the amount to use for spreading points inside a feature (currently area)
func (s *Spreader) TotalSpreadValue() float64 {
	var spreadValue float64
	for _, feat := range s.SpreadFeatures {
		spreadValue += planar.Area(feat.Geometry)
	}
	return spreadValue
}

// MakeSpreaders returns a chan of Spreader structs to efficiently process all features
func MakeSpreaders(fc *geojson.FeatureCollection, qt *quadtree.Quadtree, prop string) <-chan Spreader {
	var wg sync.WaitGroup
	features := make(chan *geojson.Feature)
	spreaders := make(chan Spreader)

	// Start up worker goroutines to process the data in goroutines so that they don't
	// block trying to read from features
	for i := 0; i < workerCount; i++ {
		go func() {
			for feat := range features {
				spreaders <- Spreader{feat, feat.Properties[prop].(float64), geom.IntersectingFeatures(qt, feat)}
				wg.Done()
			}
		}()
	}

	// Start up a goroutine that will close the spreader chan when finished
	go func() {
		defer close(spreaders)
		// Iterate through each feature, adding it to the WaitGroup and chan
		for _, feat := range fc.Features {
			wg.Add(1)
			features <- feat
		}
		close(features)
		wg.Wait()
	}()

	return spreaders
}
