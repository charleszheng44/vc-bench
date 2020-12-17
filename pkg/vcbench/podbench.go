package vcbench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
	"text/template"
	"time"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/apis"
	tenancyv1alpha1 "sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/apis/tenancy/v1alpha1"
	"sigs.k8s.io/multi-tenancy/incubator/virtualcluster/pkg/controller/secret"

	"github.com/charleszheng44/vc-bench/pkg/constants"
	"github.com/charleszheng44/vc-bench/pkg/tenant"
)

const (
	defaultPodTemp = `apiVersion: v1
kind: Pod
metadata:
  name: {{ .podname }}
  namespace: {{ .podnamespace }}
spec:
  schedulerName: kube-scheduler
  nodeSelector:
    type: virtual-kubelet
  tolerations:
  - key: virtual-kubelet.io/provider
    operator: Exists
    effect: NoSchedule
  containers:
  - name: {{ .podname }}
    image: busybox
    resources:
      requests:
        memory: "128Mi"
        cpu: "500m"
      limists:
        memory: "128Mi"
        cpu: "500m"
    command:
    - top
`
	defaultPodBaseName      = "pod"
	DefaultBenchNamespace   = "podbench"
	defaultVcKubeconfigName = "admin-kubeconfig"
	superTimeFormat         = "2020-02-20T21:12:41Z"
)

type PodBenchConfig struct {
	RsrcTemp       string
	TenantInterval int
	PodInterval    int
}

type RuntimeStatics struct {
	DwsDequeue     int
	UwsDequeue     int
	TenantCreation int
	SuperCreation  int
	SuperUpdate    int
	SuperReady     int
	PodName        string
	ClusterName    string
	PodCreated     bool
}

type BenchExecutor struct {
	client.Client
	*PodBenchConfig
	sync.Mutex

	Tenants         []tenant.Tenant
	scheme          *runtime.Scheme
	RuntimeStatics  map[string]*RuntimeStatics
	vcClients       map[string]client.Client
	waitingPodsOnVc map[string]int
}

func NewBenchExecutor(tenantsKbCfg string, tenants []tenant.Tenant, tenantInterval, podInterval, numOfVC int) (*BenchExecutor, error) {
	// generate rest.Config for accessing tenant-masters k8s
	tntKbCfgbyts, err := ioutil.ReadFile(tenantsKbCfg)
	if err != nil {
		return nil, err
	}
	tntCfg, err := clientcmd.RESTConfigFromKubeConfig(tntKbCfgbyts)
	if err != nil {
		return nil, err
	}
	be := &BenchExecutor{
		scheme:          scheme.Scheme,
		RuntimeStatics:  make(map[string]*RuntimeStatics),
		vcClients:       make(map[string]client.Client),
		waitingPodsOnVc: make(map[string]int),
		Tenants:         tenants,
		PodBenchConfig: &PodBenchConfig{
			TenantInterval: tenantInterval,
			PodInterval:    podInterval,
		},
	}

	// add Virtualcluster and ClusterVersion schemes to client
	if err := apis.AddToScheme(be.scheme); err != nil {
		return nil, err
	}
	// build client for tenant
	cli, err := client.New(tntCfg, client.Options{Scheme: be.scheme})
	if err != nil {
		return nil, err
	}
	be.Client = cli
	log.Print("built client for tenants master kube")

	// build clients for each vc
	vcLst := &tenancyv1alpha1.VirtualClusterList{}
	if err = cli.List(context.TODO(), vcLst); err != nil {
		return nil, err
	}

	if len(vcLst.Items) == 0 {
		return nil, errors.New("no virtualcluster found")
	}
	log.Printf("there are %d vc on tenants-master kube", len(vcLst.Items))
	var vcCounter int
	for _, vc := range vcLst.Items {
		vcCli, err := buildVcClient(be.Client, &vc)
		if err != nil {
			return nil, err
		}
		be.vcClients[vc.GetName()] = vcCli
		vcCounter++
		if vcCounter == numOfVC {
			break
		}
	}
	log.Printf("will use %d vc", vcCounter)
	return be, nil
}

func buildVcClient(tenantKubeCli client.Client, vc *tenancyv1alpha1.VirtualCluster) (client.Client, error) {
	rootNs := vc.Status.ClusterNamespace
	admKbCfgSrt := &v1.Secret{}
	if err := tenantKubeCli.Get(context.TODO(), types.NamespacedName{
		Namespace: rootNs,
		Name:      secret.AdminSecretName,
	}, admKbCfgSrt); err != nil {
		log.Printf("fail to get secret (%s:%s)", rootNs, secret.AdminSecretName)
		return nil, err
	}
	log.Printf("got admin-kubeconfig secret for vc(%s)", vc.GetName())
	admKbCfgBytes, exist := admKbCfgSrt.Data[secret.AdminSecretName]
	if !exist {
		return nil, fmt.Errorf("admin-kubeconfig not found for vc(%s)", vc.GetName())
	}
	// newUrl, err := benchkubeutil.GetNodePortUrl(tenantKubeCli, vc)
	// if err != nil {
	// 	return nil, err
	// }
	// externalKbCfg, err := benchkubeutil.UpdateKubeConfig(admKbCfgBytes, newUrl)
	// if err != nil {
	// 	return nil, err
	// }
	log.Printf("update admin-kubeconfig for vc(%s)", vc.GetName())
	vcRestCfg, err := clientcmd.RESTConfigFromKubeConfig(admKbCfgBytes)
	if err != nil {
		return nil, err
	}
	vcCli, err := client.New(vcRestCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}
	log.Printf("client is created for vc(%s)", vc.GetName())
	return vcCli, nil
}

func yamlBytsToObject(scheme *runtime.Scheme, yamlByts []byte) (runtime.Object, error) {
	decode := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode
	obj, _, err := decode(yamlByts, nil, nil)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (be *BenchExecutor) CleanUp(targetNs string) {
	startTime2 := time.Now().Unix()
	for vc, vcCli := range be.vcClients {
		log.Printf("will delete vk-nodes on vc %s", vc)
		if err := vcCli.DeleteAllOf(context.TODO(), &v1.Node{}, &client.DeleteAllOfOptions{}); err != nil {
			log.Printf("fail to delete nodes on vc(%s): %s", vc, err)
		}
	}
	allNodesRemoved := false
	for !allNodesRemoved {
		allNodesRemoved = true
		for vc, vcCli := range be.vcClients {
			nl := &v1.NodeList{}
			if err := vcCli.List(context.TODO(), nl, &client.ListOptions{}); err != nil {
				log.Printf("fail to list node on vc(%s): %s", vc, err)
			}
			if len(nl.Items) != 0 {
				log.Printf("there are %d nodes left on vc(%s)", len(nl.Items), vc)
				allNodesRemoved = false
			}
		}
	}
	endTime2 := time.Now().Unix()
	log.Printf("deleting all nodes took %d seconds", int(endTime2-startTime2))

	startTime := time.Now().Unix()
	deletingNamespace := 0
	for vc, vcCli := range be.vcClients {
		log.Printf("will delete namespace %s on vc %s", targetNs, vc)
		if err := vcCli.Delete(context.TODO(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: targetNs,
			},
		}); err != nil {
			log.Printf("fail to delete namespace %s on vc(%s): %s", targetNs, vc, err)
		}
		deletingNamespace++
	}
	for deletingNamespace > 0 {
		<-time.After(20 * time.Second)
		log.Printf("there are %d namespace left", deletingNamespace)
		for vc, vcCli := range be.vcClients {
			ns := v1.Namespace{}
			if err := vcCli.Get(context.TODO(), types.NamespacedName{
				Name:      DefaultBenchNamespace,
				Namespace: "default",
			}, &ns); err != nil {
				if apierrors.IsNotFound(err) {
					deletingNamespace--
					continue
				}
				log.Printf("fail to delete namespace(%s) for vc(%s)",
					DefaultBenchNamespace, vc)
				return
			} else {
				log.Printf("%s(%s) on vc(%s) is still exist", DefaultBenchNamespace, ns.Status.Phase, vc)
			}
		}
	}
	endTime := time.Now().Unix()
	log.Printf("deleting all namespaces took %d seconds", int(endTime-startTime))

}

func (be *BenchExecutor) SubmitPods(vc string, vcCli client.Client, tenant tenant.Tenant, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("[GOROUTINE] start submitting pod on vc(%s)", vc)
	benchNs := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultBenchNamespace,
		},
	}
	if err := vcCli.Create(context.TODO(), benchNs); err != nil && !apierrors.IsAlreadyExists(err) {
		log.Printf("fail to create ns(%s) on vc(%s): %s", DefaultBenchNamespace, vc, err)
		return
	}
	log.Printf("[GOROUTINE] benchmark namespace(%s) is created on vc(%s)",
		DefaultBenchNamespace, vc)
	log.Println("will sleep for 1 min to wait for sa been created")
	<-time.After(time.Duration(1) * time.Minute)
	for i := 0; i < tenant.NumPods; i++ {
		// subsitute rsrc yaml
		podName := fmt.Sprintf("%s-%s-%s%d", vc, tenant.ID, defaultPodBaseName, i)
		ctx := map[string]string{
			"podname":      podName,
			"podnamespace": DefaultBenchNamespace,
		}
		podYaml, err := fillOutTemplate(defaultPodTemp, ctx)
		if err != nil {
			log.Printf("[GOROUTINE] fail to submit pods on vc(%s): %s", vc, err)
			return
		}
		// convert yaml to object
		obj, err := yamlBytsToObject(be.scheme, podYaml)
		if err != nil {
			log.Printf("[GOROUTINE] fail to submit pods on vc(%s): %s", vc, err)
			return
		}
		pod, ok := obj.(*v1.Pod)
		if !ok {
			err = errors.New("fail to assert runtime object to pod pointer")
			log.Printf("[GOROUTINE] fail to submit pods on vc(%s): %s", vc, err)
			return
		}
		// submit rsrc
		if err = vcCli.Create(context.TODO(), pod); err != nil {
			log.Printf("[GOROUTINE] fail to submit pods on vc(%s): %s", vc, err)
			return
		}
		log.Printf("[GOROUTINE] pod(%s) created on vc(%s)", pod.GetName(), vc)

		be.Lock()
		be.RuntimeStatics[podName] = &RuntimeStatics{
			PodName:     podName,
			ClusterName: vc,
		}
		if _, exist := be.waitingPodsOnVc[vc]; !exist {
			be.waitingPodsOnVc[vc] = 1
		} else {
			be.waitingPodsOnVc[vc]++
		}
		be.Unlock()

		time.Sleep(time.Duration(be.PodInterval) * time.Second)
	}
}

func (be *BenchExecutor) RunBench() error {
	// equally spread rsrc to each vc
	var wg sync.WaitGroup
	tenantCounter := 0
	for vc, vcCli := range be.vcClients {
		if tenantCounter == len(be.Tenants) {
			break
		}
		wg.Add(1)
		go be.SubmitPods(vc, vcCli, be.Tenants[tenantCounter], &wg)
		tenantCounter++
		time.Sleep(time.Duration(be.TenantInterval) * time.Second)
	}
	log.Printf("waiting for submitting pod on vc...")
	wg.Wait()
	log.Printf("all pod submitted to vc")

	// periodically(20 seconds) check if resources are created
	for len(be.waitingPodsOnVc) > 0 {
		<-time.After(20 * time.Second)
		log.Print("checking waiting pod status")
		var remainingPods int
		for vc := range be.waitingPodsOnVc {
			log.Printf("check pods' status on vc(%s): %d pods remain", vc, be.waitingPodsOnVc[vc])
			remainingPods += be.waitingPodsOnVc[vc]
			vcCli := be.vcClients[vc]
			pl := &v1.PodList{}
			if err := vcCli.List(context.TODO(), pl, client.InNamespace(DefaultBenchNamespace)); err != nil {
				log.Printf("fail to list pods in ns(%s) of vc(%s): %s",
					DefaultBenchNamespace, vc, err)
			}
			for _, p := range pl.Items {
				dwsdq, uwsdq, tct, sct, fut, srt := getTimeInfoOnSuper(p)
				if _, exist := be.RuntimeStatics[p.GetName()]; !exist {
					be.RuntimeStatics[p.GetName()] = &RuntimeStatics{}
				}
				be.RuntimeStatics[p.GetName()].DwsDequeue = dwsdq
				be.RuntimeStatics[p.GetName()].UwsDequeue = uwsdq
				be.RuntimeStatics[p.GetName()].TenantCreation = tct
				be.RuntimeStatics[p.GetName()].SuperCreation = sct
				be.RuntimeStatics[p.GetName()].SuperUpdate = fut
				be.RuntimeStatics[p.GetName()].SuperReady = srt
				if sct != 0 && fut != 0 && be.RuntimeStatics[p.GetName()].PodCreated == false {
					// got all information, no need to poll the pod in
					// the future
					log.Printf("creation lifecycle of pod(%s) is complete", p.GetName())
					be.RuntimeStatics[p.GetName()].PodCreated = true
					be.waitingPodsOnVc[vc]--
				}
				if be.waitingPodsOnVc[vc] == 0 {
					// remove idle vc
					log.Printf("creation lifecycle of pods on vc(%s) are complete", vc)
					delete(be.waitingPodsOnVc, vc)
					break
				}
			}
		}
		log.Printf("There are %d pods remain in total", remainingPods)
	}

	// TODO clean up once done
	// delete all pods
	// log.Print("will remove all pods")
	// for p, info := range be.RuntimeStatics {
	// 	tmpPo := &v1.Pod{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Namespace: DefaultBenchNamespace,
	// 			Name:      p,
	// 		},
	// 	}
	// 	vcCli := be.vcClients[info.ClusterName]
	// 	if err := vcCli.Delete(context.TODO(), tmpPo); err != nil {
	// 		log.Printf("fail to delete po(%s) on vc(%s): %s", p, info.ClusterName, err)
	// 	}
	// }

	return nil
}

func getTimeInfoOnSuper(pod v1.Pod) (dwsdq, uwsdq, tct, sct, fut, srt int) {
	tctt := pod.GetCreationTimestamp()
	if tctt.IsZero() {
		return
	}
	tct = int(tctt.Unix())

	annos := pod.GetAnnotations()
	dwsdqStr, exist := annos[constants.LabelPerfBenchDWSReconcileTime]
	if !exist {
		return
	}
	dwsdq, err := strconv.Atoi(dwsdqStr)
	if err != nil {
		return
	}

	uwsdqStr, exist := annos[constants.LabelPerfBenchUWSReconcileTime]
	if !exist {
		return
	}
	uwsdq, err = strconv.Atoi(uwsdqStr)
	if err != nil {
		return
	}

	futStr, exist := annos[constants.LabelPerfBenchFirstUpdateTime]
	if !exist {
		return
	}
	fut, err = strconv.Atoi(futStr)
	if err != nil {
		return
	}

	sctStr, exist := annos[constants.LabelPerfBenchSuperCreationTime]
	if !exist {
		return
	}
	sct, err = strconv.Atoi(sctStr)
	if err != nil {
		return
	}

	srtStr, exist := annos[constants.LabelPerfBenchSuperReadyTime]
	if !exist {
		return
	}
	srt, err = strconv.Atoi(srtStr)
	if err != nil {
		return
	}

	return
}

func fillOutTemplate(kubeConfigTmpl string, context interface{}) ([]byte, error) {
	t, tmplPrsErr := template.New("test").Parse(kubeConfigTmpl)
	if tmplPrsErr != nil {
		return nil, tmplPrsErr
	}
	writer := bytes.NewBuffer([]byte{})
	if err := t.Execute(writer, context); nil != err {
		return nil, err
	}

	return writer.Bytes(), nil
}
