package perftimestamp

import (
	"k8s.io/api/core/v1"
	"strconv"
)

func AnnotateTimestampIfNotExist(vPod *v1.Pod, key string, timestamp int64) {
	if len(vPod.Annotations) == 0 {
		vPod.Annotations = make(map[string]string)
	}
	if _, exist := vPod.Annotations[key]; !exist {
		vPod.Annotations[key] = strconv.Itoa(int(timestamp))
	}
}

func AnnotateTimestampsIfNotExist(vPod *v1.Pod, ctx map[string]int64) {
	if len(vPod.Annotations) == 0 {
		vPod.Annotations = make(map[string]string)
	}
	for key, ts := range ctx {
		if _, exist := vPod.Annotations[key]; !exist {
			vPod.Annotations[key] = strconv.Itoa(int(ts))
		}
	}
}
