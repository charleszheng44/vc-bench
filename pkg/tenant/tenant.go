package tenant

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"
)

const tenantBaseName = "tenant"

type Tenant struct {
	ID               string `json:"id"`
	NumPods          int    `json:"numPods"`
	ConcurrentSubmit bool   `json:"concurrentSubmit"`
}

func randInRange(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

func GenRandTenantLst(numTenant int, maxPods int) []Tenant {
	var tl []Tenant
	for i := 0; i < numTenant; i++ {
		tl = append(tl, Tenant{
			ID:      tenantBaseName + "-" + strconv.Itoa(i),
			NumPods: randInRange(1, maxPods),
		})
	}
	return tl
}

func ParseTenantsJson(tenantJson string) ([]Tenant, error) {
	tjByts, err := ioutil.ReadFile(tenantJson)
	if err != nil {
		return nil, err
	}
	var tl []Tenant
	err = json.Unmarshal(tjByts, &tl)
	if err != nil {
		return nil, err
	}
	return tl, nil
}
