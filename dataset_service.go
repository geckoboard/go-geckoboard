package geckoboard

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
)

type DatasetService interface {
	FindOrCreate(context.Context, *Dataset) error
	AppendData(context.Context, *Dataset, Data) error
	ReplaceData(context.Context, *Dataset, Data) error
}

type datasetService struct {
	client           *Client
	maxRecordsPerReq int
	jsonMarshalFn    func(interface{}) ([]byte, error)
}

type DatasetType string
type FieldType string
type TimeUnit string

const (
	NumberType   FieldType = "number"
	DateType     FieldType = "date"
	DatetimeType FieldType = "datetime"
	StringType   FieldType = "string"
	PercentType  FieldType = "percentage"
	DurationType FieldType = "duration"
	MoneyType    FieldType = "money"

	Milliseconds TimeUnit = "milliseconds"
	Seconds      TimeUnit = "seconds"
	Minutes      TimeUnit = "minutes"
	Hours        TimeUnit = "hours"
)

type Dataset struct {
	Name     string           `json:"id"`
	Fields   map[string]Field `json:"fields"`
	UniqueBy []string         `json:"unique_by,omitempty"`
}

type Field struct {
	Type     FieldType `json:"type"`
	Name     string    `json:"name"`
	Optional bool      `json:"optional"`

	// Required only when field type is duration
	TimeUnit TimeUnit `json:"time_unit,omitempty"`
	// Required only when field type is money
	// ISO4217 currency code https://en.wikipedia.org/wiki/ISO_4217#Active_codes
	CurrencyCode string `json:"currency_code,omitempty"`
}

type DataRow map[string]interface{}
type Data []DataRow
type DataPayload struct {
	Data Data `json:"data"`
}

func (d *datasetService) FindOrCreate(ctx context.Context, dataset *Dataset) error {
	b, err := d.jsonMarshalFn(dataset)
	if err != nil {
		return err
	}

	req, err := d.client.buildRequest(http.MethodPut, "/datasets/"+dataset.Name, bytes.NewReader(b))
	if err != nil {
		return err
	}

	return d.client.doRequest(req.WithContext(ctx))
}

// ReplaceData replaces the existing data in the dataset, as we are limited to 500 records
// per API request - this means we can only support 500 records when replacing data
func (d *datasetService) ReplaceData(ctx context.Context, dataset *Dataset, data Data) error {
	maxRange := math.Min(float64(len(data)), float64(d.maxRecordsPerReq))
	payload := DataPayload{Data: data[0:int(maxRange)]}

	if err := d.sendData(ctx, http.MethodPut, dataset, payload); err != nil {
		return err
	}

	return nil
}

func (d *datasetService) AppendData(ctx context.Context, dataset *Dataset, data Data) error {
	grps := len(data) / d.maxRecordsPerReq
	var payload DataPayload

	for i := 0; i <= grps; i++ {
		batch := d.maxRecordsPerReq * i

		if i == grps {
			if batch+1 <= len(data) {
				payload := DataPayload{Data: data[batch:]}
				if err := d.sendData(ctx, http.MethodPost, dataset, payload); err != nil {
					return err
				}
			}
		} else {
			payload = DataPayload{Data: data[batch : d.maxRecordsPerReq*(i+1)]}
			if err := d.sendData(ctx, http.MethodPost, dataset, payload); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *datasetService) sendData(ctx context.Context, method string, dataset *Dataset, payload DataPayload) error {
	b, err := d.jsonMarshalFn(payload)
	if err != nil {
		return err
	}

	req, err := d.client.buildRequest(method, fmt.Sprintf("/datasets/%s/data", dataset.Name), bytes.NewReader(b))
	if err != nil {
		return err
	}

	return d.client.doRequest(req.WithContext(ctx))
}
