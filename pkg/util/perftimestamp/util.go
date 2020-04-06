package perftimestamp

import (
	"strconv"

	tenancyv1alpha1 "github.com/kubernetes-sigs/multi-tenancy/incubator/virtualcluster/pkg/apis/tenancy/v1alpha1"
)

func AnnotateTimestamp(vc *tenancyv1alpha1.Virtualcluster, key string, timestamp int64) {
	if len(vc.Annotations) == 0 {
		vc.Annotations = make(map[string]string)
	}
	if _, exist := vc.Annotations[key]; !exist {
		vc.Annotations[key] = strconv.Itoa(int(timestamp))
	}
}
