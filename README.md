# go-geckoboard


## Installing

Install with go modules

```
go get github.com/geckoboard/go-geckoboard
```

import with:

```go
import (
    "github.com/geckoboard/go-geckoboard
)
```

## API requests

### Find or create a new dataset

```go
	ctx := context.Background()
	svc := geckoboard.New('<apikey>').DatasetService()

	dataset := &geckoboard.Dataset{
		Name: "orders_by_country",
		Fields: map[string]geckoboard.Field{
			"report_date": {Type: geckoboard.DateType, Optional: false, Name: "Report date"},
			"country":     {Type: geckoboard.StringType, Optional: false, Name: "Country"},
			"orders":      {Type: geckoboard.NumberType, Optional: true, Name: "Orders"},
		},
		UniqueBy: []string{"report_date", "country"},
	}

	if err := svc.FindOrCreate(ctx, dataset); err != nil {
		log.Fatal(err)
	}
```

### Update a dataset

```go
err := svc.AppendData(ctx, dataset, geckoboard.Data{
	{
		"report_date": time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly),
		"country":     "United States",
		"orders":      88,
	},
	{
		"report_date": time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly),
		"country":     "United Kingdom",
		"orders":      33,
	},
	{
		"report_date": time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly),
		"country":     "United States",
		"orders":      108,
	},
	{
		"report_date": time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly),
		"country":     "United Kingdom",
		"orders":      42,
	},
})
```

### Replace all data in a dataset

```go
	err := svc.ReplaceData(ctx, dataset, geckoboard.Data{
		{
			"report_date": time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly),
			"country":     "All",
			"orders":      888,
		}
	})
```

### Delete a dataset
TBD

### Ping to test connection
TBD

## Running the tests

```
make test
```

## Still to be developed
- Support deleting a dataset
- Support appending to dataset with delete_by option
- Support ping to test connection good?
- Custom json marshaling on data to serialize date or datetime from time object based on the schema 
