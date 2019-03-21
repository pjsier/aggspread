package spreader

import (
	"math"
	"math/rand"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
	"github.com/pjsier/aggspread/pkg/geom"
)

type Spreader struct {
	Feature        *geojson.Feature
	AggregateValue float64
	SpreadFeatures []*geojson.Feature
}

const WORKER_COUNT = 8

// Return a slice of points distributed throughout spread features
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

		var i int64
		for ; i < numPoints; i++ {
			if len(spreadPoints) >= totalNumPoints {
				break
			}
			spreadPoints = append(spreadPoints, geom.RandomPointInGeom(spreadFeat.Geometry))
		}
	}

	// Randomly distribute remaining points throughout spread features, starting with largest
	// TODO: Come up with a better method that doesn't sacrifice speed
	pointsToAdd := math.Floor(s.AggregateValue - float64(len(spreadPoints)))
	for i := 0; float64(i) < pointsToAdd; i++ {
		index := int(math.Floor(rand.Float64() * float64(lenSpreadFeatures)))
		spreadPoints = append(spreadPoints, geom.RandomPointInGeom(s.SpreadFeatures[index].Geometry))
	}

	return spreadPoints
}

// Get the sum of all feature areas to know what to distribute
// TODO: Allow spreading by prop
func (s *Spreader) TotalSpreadValue() float64 {
	var spreadValue float64
	for _, feat := range s.SpreadFeatures {
		spreadValue += planar.Area(feat.Geometry)
	}
	return spreadValue
}

// Takes a feature collection of aggregate features, returns a slice of Spreader objects
// that can distribute the aggregated values throughout the spread features
func MakeSpreaders(fc *geojson.FeatureCollection, qt *quadtree.Quadtree, prop string) (int, <-chan Spreader) {
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
			spreaderChan <- Spreader{feat, feat.Properties[prop].(float64), geom.IntersectingFeatures(qt, feat)}
		}
	}()
}
