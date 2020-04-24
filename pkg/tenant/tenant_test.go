package tenant

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

const (
	TenantsJson = `
[
  {"id": "tenant1", "numPods": 1},
  {"id": "tenant2", "numPods": 5000}
]
`
	JsonFileName = "tenants.json"
)

func TestParseTenantsJson(t *testing.T) {
	if err := ioutil.WriteFile(JsonFileName, []byte(TenantsJson), 0666); err != nil {
		t.Fatalf("fail to create tmp json file")
	}
	defer os.Remove(JsonFileName)

	tl, err := ParseTenantsJson(JsonFileName)
	if err != nil {
		t.Fatalf("ParseTenantsJson failed: %s", err)
	}
	tlSample := []Tenant{
		{
			ID:      "tenant1",
			NumPods: 1,
		},
		{
			ID:      "tenant2",
			NumPods: 5000,
		},
	}
	for i, tenant := range tl {
		if !reflect.DeepEqual(tlSample[i], tenant) {
			t.Fatalf("want %v, get %v", tlSample[i], tenant)
		}
	}
}
