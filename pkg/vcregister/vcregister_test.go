package vcregister

import (
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

var (
	oldAdmKbCfg = `
kind: Config
apiVersion: v1
users:
- name: admin
  user:
    client-certificate-data: LS0XXX==
    client-key-data: LS0XXX==
clusters:
- name: vc1
  cluster:
    certificate-authority-data: LS0XXX==
    server: https://apiserver-svc.t1-vc1:6443
contexts:
- context:
    cluster: vc1
    user: admin
  name: default
current-context: default
preferences: {}
`
	newServerUrl = "https://127.0.0.1:31256"
	newAdmKbCfg  = `
kind: Config
apiVersion: v1
users:
- name: admin
  user:
    client-certificate-data: LS0XXX==
    client-key-data: LS0XXX==
clusters:
- name: vc1
  cluster:
    server: https://127.0.0.1:31256
    insecure-skip-tls-verify: true
contexts:
- context:
    cluster: vc1
    user: admin
  name: default
current-context: default
preferences: {}
`
)

func TestUpdateKubeConfigSecret(t *testing.T) {
	resultCfg, err := updateKubeConfig([]byte(oldAdmKbCfg), newServerUrl)
	if err != nil {
		t.Fatalf("updateKubeConfig fail: %s", err)
	}

	if yamlDeepEqual(string(resultCfg), newAdmKbCfg) == false {
		t.Fatalf("updateKubeConfig fail: want %s get %s", newAdmKbCfg, resultCfg)
	}
}

func yamlDeepEqual(yamlStr1, yamlStr2 string) bool {
	ym1 := make(map[interface{}]interface{})
	ym2 := make(map[interface{}]interface{})
	if err := yaml.Unmarshal([]byte(yamlStr1), ym1); err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal([]byte(yamlStr2), ym2); err != nil {
		panic(err)
	}
	return reflect.DeepEqual(ym1, ym2)
}
