module github.com/charleszheng44/vc-bench

go 1.14

require k8s.io/kubernetes v1.16.5

replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.7-beta.0
	k8s.io/apiserver => k8s.io/apiserver v0.16.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.7-beta.0
	k8s.io/component-base => k8s.io/component-base v0.16.5
	k8s.io/cri-api => k8s.io/cri-api v0.16.8-beta.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.6-0.20200117003438-6c2bc98ccc12
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20200117000206-3f805ead6cdb
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.6-0.20200117002812-f8a6ab64503b
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.6-0.20200117002145-8ee09a1df25e
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.5
	k8s.io/kubectl => k8s.io/kubectl v0.16.6-0.20200117004440-8c0ffec5adaf
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20200117002354-7275ea7ddf81
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190819145509-592c9a46fd00
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190819143841-305e1cef1ab1
	k8s.io/node-api => k8s.io/node-api v0.0.0-20190819145652-b61681edbd0a
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190819143045-c84c31c165c4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20200117001934-76e3585bc069
	k8s.io/sample-controller => k8s.io/sample-controller v0.16.6-0.20200117000807-3588e4c3da81
)
