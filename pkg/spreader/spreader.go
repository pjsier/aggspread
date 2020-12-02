package spreader

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strconv"
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
		portion := geom.OverlapArea(s.Feature.Geometry, spreadFeat.Geometry) / spreadTotal
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
		spreadValue += geom.OverlapArea(s.Feature.Geometry, feat.Geometry)
	}
	return spreadValue
}

func getFloat(val interface{}) (float64, error) {
	switch i := val.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint:
		return float64(i), nil
	case string:
		return strconv.ParseFloat(i, 64)
	default:
		return math.NaN(), fmt.Errorf("Cannot parse value of type '%s'", i)
	}
}

// NewSpreader creates a Spreader, returning nil if the property value cannot be parsed
func NewSpreader(feat *geojson.Feature, features []*geojson.Feature, prop string) (*Spreader, error) {
	propVal, err := getFloat(feat.Properties[prop])
	if err != nil {
		return nil, fmt.Errorf("Could not parse value '%s' as float64", feat.Properties[prop])
	}
	return &Spreader{feat, propVal, features}, nil
}

// MakeSpreaders returns a chan of Spreader structs to efficiently process all features
func MakeSpreaders(fc *geojson.FeatureCollection, prop string, qt *quadtree.Quadtree) <-chan *Spreader {
	var wg sync.WaitGroup
	features := make(chan *geojson.Feature)
	spreaders := make(chan *Spreader)

	// Start up worker goroutines to process the data in goroutines so that they don't
	// block trying to read from features
	for i := 0; i < workerCount; i++ {
		go func() {
			for feat := range features {
				// Default to spreading over the input feature if a quadtree not supplied
				var intersectingFeatures []*geojson.Feature
				if qt == nil {
					intersectingFeatures = []*geojson.Feature{feat}
				} else {
					intersectingFeatures = geom.IntersectingFeatures(qt, feat)
				}
				spreader, err := NewSpreader(feat, intersectingFeatures, prop)
				if err != nil {
					log.Printf("Cannot spread feature due to error: %s", err)
					wg.Done()
					continue
				}
				spreaders <- spreader
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
