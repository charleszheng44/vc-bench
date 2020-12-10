package vcregister

import (
	"context"
	"errors"
	"fmt"
	"log"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	benchkubeutil "github.com/charleszheng44/vc-bench/pkg/util/kube"
	benchvcutil "github.com/charleszheng44/vc-bench/pkg/util/vc"
	"sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/apis"
	tenancyv1alpha1 "sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/apis/tenancy/v1alpha1"
	"sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/controller/secret"
	"sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/syncer/constants"
)

const (
	controllerName = "VirtualclusterRegister"
)

var vcGVK schema.GroupVersionKind = schema.GroupVersionKind{
	Group:   "tenancy.x-k8s.io",
	Version: "v1alpha1",
	Kind:    "VirtualCluster",
}

// VirtualclusterRegister is the controller syncs the Virtualcluster(VC) cr
// between the metaCluster and the tenantCluster (k8s that hosts vc masters)
type VirtualclusterRegister struct {
	// tenantClusterCache contains informers that monitor resources
	// (i.e. Virtualcluster) on cluster that manages tenant masters
	tenantClusterCache cache.Cache
	// directly read object from the apiserver
	tenantClusterReader client.Reader
	// metaClusterClient CRUDs Virtualcluster on metaCluster
	metaClusterClient client.Client
	// scheme is shared by tenantClusterCache and metaClusterClient
	scheme *runtime.Scheme
}

// New initializes a VirtualclusterRegister with metaClusterClient and
// tenantClusterCache share a runtime.Scheme
func New(metaKbCfg, tcKbCfg string) (*VirtualclusterRegister, error) {
	// tcKbCfg cannot be none
	if tcKbCfg == "" {
		return nil,
			errors.New("path to kubeconfig of tenant-masters management cluster cannot be empty")
	}

	// metaKbCfg and tcKbCfg can not be the same
	if metaKbCfg == tcKbCfg {
		return nil,
			errors.New("meta kubeconfig is same as tenant-masters kubeconfig")
	}

	// 0. add Virtualcluster and ClusterVersion GVK to scheme
	registerScheme := scheme.Scheme
	if err := apis.AddToScheme(registerScheme); err != nil {
		return nil, err
	}

	// 1. create VirtualclusterRegister and initialize an empty scheme
	// NOTE metaClusterClient and tenantClusterCache will share the same scheme
	vcr := &VirtualclusterRegister{scheme: registerScheme}

	// 2. create the metaClusterClient
	var (
		metaCfg    *rest.Config
		metaCfgErr error
	)
	if metaKbCfg == "" {
		metaCfg, metaCfgErr = rest.InClusterConfig()
		if metaCfgErr != nil {
			return nil, metaCfgErr
		}
	} else {
		metaCfg, metaCfgErr = clientcmd.BuildConfigFromFlags("", metaKbCfg)
		if metaCfgErr != nil {
			return nil, metaCfgErr
		}
	}

	metaClusterClient, err := client.New(metaCfg, client.Options{Scheme: vcr.scheme})
	if err != nil {
		return nil, err
	}
	vcr.metaClusterClient = metaClusterClient

	// 3. create the tenant cluster reader
	tcCfg, err := clientcmd.BuildConfigFromFlags("", tcKbCfg)
	tenantClusterReader, err := client.New(tcCfg, client.Options{Scheme: vcr.scheme})
	if err != nil {
		return nil, err
	}
	vcr.tenantClusterReader = tenantClusterReader

	// 4. build the cache for tenant cluster
	tenantClusterCache, err := cache.New(tcCfg, cache.Options{Scheme: vcr.scheme})
	if err != nil {
		return nil, err
	}
	vcr.tenantClusterCache = tenantClusterCache

	return vcr, nil
}

// Start starts the VirtualclusterRegister by first adding the crd to
// the scheme, then registering event handlers for informers in
// tenantClusterCache, and run the informer.
func (vcr *VirtualclusterRegister) Start(stop <-chan struct{}) error {
	// get vc informer, if the informer not exist
	// `GetInformerForKind(gvk schema.GroupVersionKind)` will
	// create one
	vcIfm, err := vcr.tenantClusterCache.GetInformerForKind(context.TODO(), vcGVK)
	if err != nil {
		return err
	}

	// 2. add event handler
	vcIfm.AddEventHandler(
		toolscache.ResourceEventHandlerFuncs{
			AddFunc:    vcr.onAdd,
			UpdateFunc: vcr.onUpdate,
			DeleteFunc: vcr.onDelete,
		})

	errChan := make(chan error)
	defer close(errChan)

	// 3. start the cache, which will also run the informer
	go func() {
		log.Print("starting the tenantClusterCache")
		if err := vcr.tenantClusterCache.Start(stop); err != nil {
			errChan <- err
		}
	}()

	// sync the cache
	if ok := vcr.tenantClusterCache.WaitForCacheSync(stop); !ok {
		return errors.New("failed to sync tenant-cluster cache")
	}

	select {
	case retErr := <-errChan:
		log.Printf("failed with error: %s", retErr)
		return retErr
	case <-stop:
		log.Print("stopped")
	}
	return nil
}

// onAdd registers obj on the meta cluster
func (vcr *VirtualclusterRegister) onAdd(obj interface{}) {
	vc, ok := obj.(*tenancyv1alpha1.VirtualCluster)
	if !ok {
		log.Print("onAdd function fail to convert object to Virtualcluster")
	}
	log.Printf("adding Virtualcluster(%s) to meta cluster with uid: %s", vc.GetName(), vc.GetUID())

	vc = filterVcMeta(vc)

	// 1. create the tenant namespace
	if err := benchkubeutil.CreateNS(vcr.metaClusterClient, vc.GetNamespace()); err != nil {
		log.Printf("fail to create tenant namespace(%s) for virtualcluster(%s)",
			vc.GetNamespace(), vc.GetName())
	}

	// 2. if vc is running, create the root ns and copy the admin-kubeconfig
	if vc.Status.Phase == tenancyv1alpha1.ClusterRunning {
		vcr.initializeVcOnMeta(vc)
	}

	// 3. create the vc
	if err := vcr.metaClusterClient.Create(context.TODO(), vc); err != nil {
		log.Printf("fail to adding Virtualcluster(%s) to meta cluster: %s",
			vc.Name, err)
	}
	log.Printf("Virtualcluster(%s) is created on meta cluster with uid: %s", vc.GetName(), vc.GetUID())
}

func filterObjMeta(objMeta metav1.ObjectMeta) metav1.ObjectMeta {
	objMeta.ResourceVersion = ""
	// objMeta.UID = types.UID("")
	return objMeta
}

func filterVcMeta(vc *tenancyv1alpha1.VirtualCluster) *tenancyv1alpha1.VirtualCluster {
	vc.ObjectMeta = filterObjMeta(vc.ObjectMeta)
	vc.ObjectMeta.Finalizers = []string{}
	return vc
}

func (vcr *VirtualclusterRegister) initializeVcOnMeta(vc *tenancyv1alpha1.VirtualCluster) {
	log.Printf("initialzing vc(%s) on super", vc.GetName())
	// 1. update the admin-kubeconfig
	rootNs := benchvcutil.ToClusterKey2(vc)
	admKbCfgSrt := &v1.Secret{}
	if err := vcr.tenantClusterReader.Get(context.TODO(), types.NamespacedName{
		Namespace: rootNs,
		Name:      secret.AdminSecretName,
	}, admKbCfgSrt); err != nil {
		log.Printf("fail to get secret (%s:%s)", rootNs, secret.AdminSecretName)
		return
	}
	// 2. replacing the server's url in admin-kubeconfig
	// [SYNC_PERF_2] We are using the hostnetwork now, no need to update the serverurl
	//
	// newUrl, getIPErr := benchkubeutil.GetNodePortUrl(vcr.tenantClusterReader, vc)
	// if getIPErr != nil {
	// 	log.Printf("fail to get node IP for Virtualcluster(%s): %s", vc.GetName(), getIPErr)
	// 	return
	// }
	// admKbCfgSrt, updateErr := updateKubeConfigSecret(admKbCfgSrt, newUrl)
	// if updateErr != nil {
	// 	log.Printf("fail to update admin-kubeconfig secret for Virtualcluster(%s): %s",
	// 		vc.GetName(), updateErr)
	// 	return
	// }

	// 3. create the root namespace of the vc
	if err := benchkubeutil.CreateNS(vcr.metaClusterClient, rootNs); err != nil {
		log.Printf("fail to create the root namespace(%s) for virtualcluster(%s): %s",
			rootNs, vc.GetName(), err)
		return
	}
	// 4. create a secret that contains the admin-kubeconfig
	admKbCfgSrt.ObjectMeta = filterObjMeta(admKbCfgSrt.ObjectMeta)
	if err := vcr.metaClusterClient.Create(context.TODO(), admKbCfgSrt); err != nil {
		log.Printf("fail to create secret(%s:%s) on meta cluster: %s",
			rootNs, secret.AdminSecretName, err)
		return
	}
}

// onUpdate updates new object on the meta cluster
func (vcr *VirtualclusterRegister) onUpdate(old, new interface{}) {
	oldVc, ok := old.(*tenancyv1alpha1.VirtualCluster)
	if !ok {
		log.Printf("onUpdate function fail to convert old object to Virtualcluster")
		return
	}
	vc, ok := new.(*tenancyv1alpha1.VirtualCluster)
	if !ok {
		log.Printf("onUpdate function fail to convert new object to Virtualcluster")
		return
	}
	log.Printf("updating Virtualcluster(%s) on meta cluster", vc.Name)

	// get corresponding vc on meta cluster
	metaVc := &tenancyv1alpha1.VirtualCluster{}
	if err := vcr.metaClusterClient.Get(context.TODO(), types.NamespacedName{
		Namespace: vc.GetNamespace(),
		Name:      vc.GetName(),
	}, metaVc); err != nil {
		log.Printf("fail to get Virtualcluster(%s) on meta cluster", metaVc.GetName())
		return
	}
	vc = filterVcMeta(vc)
	vc.ObjectMeta.UID = types.UID("")
	vc.ResourceVersion = metaVc.GetResourceVersion()
	vc.DeletionTimestamp = metaVc.GetDeletionTimestamp()

	if vc.Status.Phase == tenancyv1alpha1.ClusterRunning &&
		oldVc.Status.Phase == tenancyv1alpha1.ClusterPending {
		vcr.initializeVcOnMeta(vc)
	}

	if err := vcr.metaClusterClient.Update(context.TODO(), vc); err != nil {
		log.Printf("fail to update Virtualcluster(%s) on meta cluster: %s",
			metaVc.GetName(), err)
	}
}

func updateKubeConfigSecret(oldSrt *v1.Secret, newServerUrl string) (*v1.Secret, error) {
	admKbCfgByts, exist := oldSrt.Data[secret.AdminSecretName]
	if !exist {
		return nil, fmt.Errorf("Secret(%s) doesn't contain Data[%s]",
			oldSrt.GetName(), secret.AdminSecretName)
	}

	newAdmKbCfgByts, err := benchkubeutil.UpdateKubeConfig(admKbCfgByts, newServerUrl)
	if err != nil {
		return nil, err
	}
	// assign it back
	oldSrt.Data[secret.AdminSecretName] = newAdmKbCfgByts
	return oldSrt, nil
}

func containsString(strLst []string, str string) bool {
	for _, s := range strLst {
		if s == str {
			return true
		}
	}
	return false
}

func removesString(strLst []string, str string) []string {
	var retLst []string
	for i := range strLst {
		if strLst[i] != str {
			retLst = append(retLst, strLst[i])
		}
	}
	return retLst
}

// isRelatedToVc checks if given namespace is related to vc, i.e. contains
// annotation "tenancy.x-k8s.io/cluster":conversion.ToClusterKey(vc)
// NOTE the root ns doesn't contain this annotation
func isRelatedToVc(ns *v1.Namespace, vc *tenancyv1alpha1.VirtualCluster) bool {
	var tmpVcName string
	for k, v := range ns.GetAnnotations() {
		if k == constants.LabelCluster {
			tmpVcName = v
			break
		}
	}
	if tmpVcName != "" && tmpVcName == benchvcutil.ToClusterKey2(vc) {
		return true
	}
	return false
}

func (vcr *VirtualclusterRegister) deleteBelongings(vc *tenancyv1alpha1.VirtualCluster) {
	// delete all related ns, except the root ns
	nsLst := &v1.NamespaceList{}
	if err := vcr.metaClusterClient.List(context.TODO(), nsLst); err != nil {
		log.Printf("fail to get all namespaces: %s", err)
		return
	}
	for _, ns := range nsLst.Items {
		if isRelatedToVc(&ns, vc) {
			// remove related ns
			if err := vcr.metaClusterClient.Delete(context.TODO(), &ns); err != nil && !apierrors.IsNotFound(err) {
				log.Printf("fail to delete namespace: %s", ns.GetName())
				return
			}
			log.Printf("namespace(%s) is deleted", ns.GetName())
		}
	}
	// delete the tenant ns
	if err := vcr.metaClusterClient.Delete(context.TODO(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: vc.GetNamespace(),
		},
	}); err != nil && !apierrors.IsNotFound(err) {
		log.Printf("fail to delete the tenant namespace(%s): %s",
			vc.GetNamespace(), err)
		return
	}
	log.Printf("tenant namespace(%s) is deleted", vc.GetNamespace())

}

// onDelete removes obj from the meta cluster
func (vcr *VirtualclusterRegister) onDelete(obj interface{}) {
	vc, ok := obj.(*tenancyv1alpha1.VirtualCluster)
	if !ok {
		log.Print("onDelete function fail to convert object to Virtualcluster")
	}
	vcr.deleteBelongings(vc)
	if err := vcr.metaClusterClient.Delete(context.TODO(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: benchvcutil.ToClusterKey2(vc)},
	}); err != nil && !apierrors.IsNotFound(err) {
		log.Printf("fail to delete the root namespace: %s", benchvcutil.ToClusterKey2(vc))
		return
	}
	vc = filterVcMeta(vc)
	if err := vcr.metaClusterClient.Delete(context.TODO(), vc); err != nil {
		log.Printf("failed to remove Virtualcluster(%s) from the meta cluster: %s",
			vc.Name, err)
	}
	log.Printf("Virtualcluster(%s) has been removed", vc.GetName())
}
