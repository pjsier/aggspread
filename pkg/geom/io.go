package geom

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/paulmach/orb/geojson"
	"github.com/pkg/errors"
)

// GeoJSONValue is an intermediate struct for unmarshalling one record at a time so that
// we can load a FeatureCollection while ignoring invalid Features
type FeatureCollectionValue struct {
	Features []json.RawMessage `json:"features"`
}

// ParseFeatureCollection unmarshals features individually to avoid failing on invalid features
func ParseFeatureCollection(data []byte) (*geojson.FeatureCollection, error) {
	var featureCollection = geojson.NewFeatureCollection()
	var featureVal = &FeatureCollectionValue{}

	err := json.Unmarshal(data, featureVal)
	if err != nil {
		return featureCollection, err
	}

	for _, rawFeat := range featureVal.Features {
		feat, err := geojson.UnmarshalFeature(rawFeat)
		if err != nil || feat.Geometry == nil {
			log.Printf("Error occurred parsing feature, continuing: %s", err)
			continue
		}
		featureCollection.Append(feat)
	}

	return featureCollection, nil
}

// LoadGeoJSONFile accepts a filename and returns the output of parsing a FeatureCollection
func LoadGeoJSONFile(filename string) (*geojson.FeatureCollection, error) {
	var data []byte
	var err error

	if filename == "" {
		return nil, errors.New("Filename must not be blank")
	}

	if filename == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(filename)
	}

	if err != nil {
		return nil, err
	}

	return ParseFeatureCollection(data)
}
