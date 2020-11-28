package geom

import (
	"testing"
)

func TestParseFeatureCollection(t *testing.T) {
	data := []byte(`
	{
		"features": [
			{
				"type": "Feature",
				"geometry": null,
				"properties": {"test": 1}
			},
			{
				"type": "Feature",
				"properties": {"test": 2},
				"geometry": {
					"type": "Polygon",
					"coordinates": [
						[ [100.0, 0.0], [101.0, 0.0], [101.0, 1.0],
						[100.0, 1.0], [100.0, 0.0] ]
					]
				}
			}
		]
	}
	`)
	fc, err := ParseFeatureCollection(data)
	if err != nil {
		t.Errorf("Feature collection should not fail on invalid records, failed with: %e", err)
	}
	if len(fc.Features) != 1 {
		t.Errorf("Feature collection should only include valid records, length was %d", len(fc.Features))
	}
}
