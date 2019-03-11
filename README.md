# aggspread

A CLI tool based on Conveyal's [`aggregate-disser`](https://github.com/conveyal/aggregate-disser/) for spreading aggregated GeoJSON feature data throughout points inside overlapping spread features. An example would be distributing population data from Census block groups into random points within contained residential parcels.

## Running

```bash
./aggspread -agg <AGGREGATED_GEOJSON> -spread <SPREAD_GEOJSON> -prop <AGGREGATE_PROP> -output <OUTPUT_CSV_FILE>
```

## Example

Convert a feature collection of voting precincts with a property indicating the number of votes into points spread throughout residential parcels within each precinct.

![Map screenshots of aggregated and spread data](./img/example.jpg "Map screenshots of aggregated and spread data")
