package vc

import (
	"crypto/sha256"
	"encoding/hex"

	"sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/apis/tenancy/v1alpha1"
)

func ToClusterKey2(vc *v1alpha1.VirtualCluster) string {
	annos := vc.GetAnnotations()
	uid := annos["tenant-uid"]
	digest := sha256.Sum256([]byte(uid))
	return vc.GetNamespace() + "-" + hex.EncodeToString(digest[0:])[0:6] + "-" + vc.GetName()
}
