package geom

import (
	"io/ioutil"
	"os"

	"github.com/paulmach/orb/geojson"
	"github.com/pkg/errors"
)

// LoadGeoJSONFile accepts a filename and returns the output of parsing a FeatureCollection
func LoadGeoJSONFile(filename string) (*geojson.FeatureCollection, error) {
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
