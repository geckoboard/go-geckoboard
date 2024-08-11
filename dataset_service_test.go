package geckoboard

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

func TestDatasetService_FindOrCreate(t *testing.T) {
	t.Run("successfully creates dataset", func(t *testing.T) {
		want := &Dataset{
			Name: "bullhorn-test",
			Fields: map[string]Field{
				"id": {
					Name:     "ID",
					Type:     StringType,
					Optional: false,
				},
				"created_at": {
					Name:     "Created at",
					Type:     DatetimeType,
					Optional: true,
				},
				"index": {
					Name:     "Index",
					Type:     NumberType,
					Optional: true,
				},
				"money": {
					Name:         "Money",
					Type:         MoneyType,
					CurrencyCode: "USD",
					Optional:     true,
				},
				"duration": {
					Name:     "Time taken",
					Type:     MoneyType,
					TimeUnit: Hours,
					Optional: true,
				},
				"percent": {
					Name: "Completed %",
					Type: PercentType,
				},
			},
			UniqueBy: []string{"id"},
		}

		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, r.Header.Get("Authorization"), "Basic a2V5LTQ0NDo=")

			payload := map[string]any{}
			assert.NilError(t, json.NewDecoder(r.Body).Decode(&payload))

			// Validate keys omitted that shouldn't exist
			assert.DeepEqual(t, payload, map[string]any{
				"fields": map[string]any{
					"created_at": map[string]any{
						"name":     "Created at",
						"optional": true,
						"type":     "datetime",
					},
					"duration": map[string]any{
						"name":      "Time taken",
						"optional":  true,
						"time_unit": "hours",
						"type":      "money",
					},
					"id": map[string]any{
						"name":     "ID",
						"optional": false,
						"type":     "string",
					},
					"money": map[string]any{
						"currency_code": "USD",
						"name":          "Money",
						"optional":      true,
						"type":          "money",
					},
					"index": map[string]any{
						"name":     "Index",
						"optional": true,
						"type":     "number",
					},
					"percent": map[string]any{
						"name":     "Completed %",
						"optional": false,
						"type":     "percentage",
					},
				},
				"id":        "bullhorn-test",
				"unique_by": []any{"id"},
			})

			w.WriteHeader(http.StatusOK)
		})
		defer server.Close()

		ds := newService(server.URL)
		assert.NilError(t, ds.FindOrCreate(context.Background(), want))
	})

	t.Run("returns error when marshaling body fails", func(t *testing.T) {
		ds := &datasetService{
			client: NewWithURL("key-444", ""),
			jsonMarshalFn: func(any) ([]byte, error) {
				return nil, errors.New("marshal error")
			},
		}
		err := ds.FindOrCreate(context.Background(), &Dataset{})
		assert.ErrorContains(t, err, "marshal error")
	})

	t.Run("returns error with invalid url", func(t *testing.T) {
		ds := newService(string([]byte{0x7f}))
		err := ds.FindOrCreate(context.Background(), &Dataset{})
		assert.ErrorContains(t, err, "invalid control character in URL")
	})

	t.Run("returns error when request fails", func(t *testing.T) {
		err := newService("").FindOrCreate(context.Background(), &Dataset{})
		assert.ErrorContains(t, err, "unsupported protocol scheme")
	})

	t.Run("returns error when response 500", func(t *testing.T) {
		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer server.Close()

		err := newService(server.URL).FindOrCreate(context.Background(), &Dataset{})
		assert.Error(t, err, errUnexpectedResponse.Error())
	})

	t.Run("returns geckoboard error when response 400", func(t *testing.T) {
		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":{"message": "invalid field type"}}`)
		})
		defer server.Close()

		err := newService(server.URL).FindOrCreate(context.Background(), &Dataset{})
		assert.DeepEqual(t, err, &Error{
			StatusCode: http.StatusBadRequest,
			Detail: Detail{
				Message: "invalid field type",
			},
		})
	})

	t.Run("returns err when it fail to parse error json", func(t *testing.T) {
		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{invalid json}`)
		})
		defer server.Close()

		err := newService(server.URL).FindOrCreate(context.Background(), &Dataset{})
		assert.ErrorType(t, err, &json.SyntaxError{})
	})
}

func TestDatasetService_AppendData(t *testing.T) {
	t.Run("makes a single data request", func(t *testing.T) {
		var requests int

		wantData := Data{
			{
				"id":         "1234",
				"title":      "My title",
				"created_at": "2022-05-10T11:12:13Z",
			},
		}

		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, http.MethodPost)
			assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, r.Header.Get("Authorization"), "Basic a2V5LTQ0NDo=")
			requests += 1

			got := &DataPayload{}
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Fatal(err)
			}

			assert.DeepEqual(t, got, &DataPayload{Data: wantData})
			w.WriteHeader(http.StatusNoContent)
		})
		defer server.Close()

		ds := newService(server.URL)
		ds.maxRecordsPerReq = 500

		err := ds.AppendData(context.Background(), &Dataset{Name: "test-dataset"}, wantData)
		assert.NilError(t, err)
		assert.Equal(t, requests, 1)
	})

	t.Run("makes a multiple data requests", func(t *testing.T) {
		var requests int
		wantData := Data{
			{"id": "3333", "title": "title one", "created_at": "2022-05-10T11:12:13Z"},
			{"id": "2222", "title": "title two", "created_at": "2022-02-10T11:12:13Z"},
			{"id": "1111", "title": "title three", "created_at": "2022-01-10T11:12:13Z"},
			{"id": "750", "title": "title four", "created_at": "2021-11-22T11:12:13Z"},
			{"id": "555", "title": "title five", "created_at": "2021-12-15T11:12:13Z"},
		}

		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, r.Header.Get("Authorization"), "Basic a2V5LTQ0NDo=")
			requests += 1

			got := &DataPayload{}
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Fatal(err)
			}

			switch requests {
			case 1:
				assert.DeepEqual(t, got, &DataPayload{Data: wantData[0:2]})
			case 2:
				assert.DeepEqual(t, got, &DataPayload{Data: wantData[2:4]})
			case 3:
				assert.DeepEqual(t, got, &DataPayload{Data: wantData[4:]})
			}

			w.WriteHeader(http.StatusNoContent)
		})
		defer server.Close()

		ds := newService(server.URL)
		ds.maxRecordsPerReq = 2

		ctx := context.Background()
		err := ds.AppendData(ctx, &Dataset{Name: "test-dataset"}, wantData)
		assert.NilError(t, err)
		assert.Equal(t, requests, 3)
	})

	t.Run("returns error when request body marshal fails", func(t *testing.T) {
		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {})
		defer server.Close()

		ds := newService(server.URL)
		ds.jsonMarshalFn = func(any) ([]byte, error) {
			return nil, errors.New("marshal error")
		}
		ds.maxRecordsPerReq = 2

		ctx := context.Background()
		err := ds.AppendData(ctx, &Dataset{Name: "test-dataset"}, Data{{}})
		assert.ErrorContains(t, err, "marshal error")
	})

	t.Run("returns error when building the request fails", func(t *testing.T) {
		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {})
		defer server.Close()

		ctx := context.Background()
		ds := newService(string([]byte{0x7f}))
		ds.maxRecordsPerReq = 1

		err := ds.AppendData(ctx, &Dataset{Name: "test-dataset"}, Data{{}, {}})
		assert.ErrorContains(t, err, "invalid control character in URL")
	})
}

func TestDatasetService_ReplaceData(t *testing.T) {
	t.Run("makes a single data request of the first max records per request", func(t *testing.T) {
		var requests int

		dataIn := Data{
			{"id": "123", "title": "My title 1", "created_at": "2022-05-10T11:12:13Z"},
			{"id": "345", "title": "My title 2", "created_at": "2022-05-10T11:12:13Z"},
			{"id": "567", "title": "My title 3", "created_at": "2022-05-10T11:12:13Z"},
			{"id": "789", "title": "My title 4", "created_at": "2022-05-10T11:12:13Z"},
		}

		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, http.MethodPut)
			assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, r.Header.Get("Authorization"), "Basic a2V5LTQ0NDo=")
			requests += 1

			got := &DataPayload{}
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Fatal(err)
			}

			assert.DeepEqual(t, got, &DataPayload{Data: dataIn[0:3]})
			w.WriteHeader(http.StatusNoContent)
		})
		defer server.Close()

		ds := newService(server.URL)
		ds.maxRecordsPerReq = 3

		err := ds.ReplaceData(context.Background(), &Dataset{Name: "test-dataset"}, dataIn)
		assert.NilError(t, err)
		assert.Equal(t, requests, 1)
	})

	t.Run("fetches only the maximum number of slice items possible", func(t *testing.T) {
		dataIn := Data{
			{"id": "123", "title": "My title 1", "created_at": "2022-05-10T11:12:13Z"},
			{"id": "345", "title": "My title 2", "created_at": "2022-05-10T11:12:13Z"},
			{"id": "567", "title": "My title 3", "created_at": "2022-05-10T11:12:13Z"},
			{"id": "789", "title": "My title 4", "created_at": "2022-05-10T11:12:13Z"},
		}

		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {
			got := &DataPayload{}
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Fatal(err)
			}

			assert.DeepEqual(t, got, &DataPayload{Data: dataIn})
			w.WriteHeader(http.StatusNoContent)
		})
		defer server.Close()

		ds := newService(server.URL)
		ds.maxRecordsPerReq = 10

		// This could panic if we tried to do data[0:10] so it should select [0:4] as that
		// is the maximum number of records
		err := ds.ReplaceData(context.Background(), &Dataset{Name: "test-dataset"}, dataIn)
		assert.NilError(t, err)
	})

	t.Run("returns error when request body marshal fails", func(t *testing.T) {
		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {})
		defer server.Close()

		ds := newService(server.URL)
		ds.jsonMarshalFn = func(any) ([]byte, error) {
			return nil, errors.New("marshal error")
		}
		ds.maxRecordsPerReq = 2

		ctx := context.Background()
		err := ds.ReplaceData(ctx, &Dataset{Name: "test-dataset"}, Data{{}})
		assert.ErrorContains(t, err, "marshal error")
	})

	t.Run("returns error when building the request fails", func(t *testing.T) {
		server := buildMockServer(func(w http.ResponseWriter, r *http.Request) {})
		defer server.Close()

		ctx := context.Background()
		ds := newService(string([]byte{0x7f}))
		ds.maxRecordsPerReq = 1

		err := ds.ReplaceData(ctx, &Dataset{Name: "test-dataset"}, Data{{}, {}})
		assert.ErrorContains(t, err, "invalid control character in URL")
	})
}

func newService(url string) *datasetService {
	return &datasetService{
		client:           NewWithURL("key-444", url),
		jsonMarshalFn:    json.Marshal,
		maxRecordsPerReq: 500,
	}
}

func buildMockServer(handlerFn func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handlerFn))
}
