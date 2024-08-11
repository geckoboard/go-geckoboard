package geckoboard

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestGeckoboardClient_New(t *testing.T) {
	c := New("apikey-1245")

	assert.Equal(t, c.apiKey, "apikey-1245")
	assert.Equal(t, c.baseURL, "https://api.geckoboard.com")
	assert.Assert(t, c.client != nil)

	ds := c.DatasetService().(*datasetService)
	assert.Equal(t, ds.client, c)
	assert.Equal(t, ds.maxRecordsPerReq, 500)
}

func TestGeckoboardClient_NewWithURL(t *testing.T) {
	c := NewWithURL("https://example.com", "apikey-1245")

	assert.Equal(t, c.apiKey, "apikey-1245")
	assert.Equal(t, c.baseURL, "https://example.com")
	assert.Assert(t, c.client != nil)

	ds := c.DatasetService().(*datasetService)
	assert.Equal(t, ds.client, c)
	assert.Equal(t, ds.maxRecordsPerReq, 500)
}
