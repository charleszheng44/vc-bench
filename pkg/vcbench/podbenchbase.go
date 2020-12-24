package vcbench

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BasePodStatiscs struct {
	CreationTimestamp int64
	ReadyTimestamp    int64
}

type BaseBenchExecutor struct {
	CliLst         []client.Client
	NumPod         int
	NumTenants     int
	PodInterval    int
	RuntimeStatics map[string]*BasePodStatiscs
	ShareNamespace bool
}

func NewBaseBenchExecutor(kubeconfigPath string, numPod, podInterval, numTenants int, shareNs bool) (*BaseBenchExecutor, error) {
	if numPod < numTenants {
		return nil, fmt.Errorf("numPod(%d) and numTenant(%d) is wrong", numPod, numTenants)
	}

	log.Printf("There are %d pods and %d tenants", numPod, numTenants)

	kbCfgByts, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	kbCfg, err := clientcmd.RESTConfigFromKubeConfig(kbCfgByts)
	if err != nil {
		return nil, err
	}
	cliLst := []client.Client{}
	for i := 0; i < numTenants; i++ {
		kbCfg.QPS = 500
		kbCfg.Burst = 1000
		cli, err := client.New(kbCfg, client.Options{Scheme: scheme.Scheme})
		if err != nil {
			return nil, err
		}
		cliLst = append(cliLst, cli)
	}

	return &BaseBenchExecutor{
		CliLst:         cliLst,
		NumTenants:     numTenants,
		NumPod:         numPod,
		PodInterval:    podInterval,
		RuntimeStatics: make(map[string]*BasePodStatiscs),
		ShareNamespace: shareNs,
	}, nil
}

func (bbe *BaseBenchExecutor) SubmitPods(cli client.Client, numPod, tenantId int, wg *sync.WaitGroup) error {
	defer wg.Done()
	if !bbe.ShareNamespace {
		// create namespace
		tenantNs := fmt.Sprintf("%s-%d", DefaultBenchNamespace, tenantId)
		if err := cli.Create(context.TODO(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: tenantNs,
			},
		}); err != nil {
			return err
		}
		log.Printf("benchmark namespace %s is created", tenantNs)
	}
	for i := 1; i <= numPod; i++ {
		podName := fmt.Sprintf("%s%d", defaultPodBaseName, i)
		ctx := map[string]string{
			"podname":      podName,
			"podnamespace": fmt.Sprintf("%s-%d", DefaultBenchNamespace, tenantId),
		}
		if bbe.ShareNamespace {
			ctx["podname"] = fmt.Sprintf("tenant%d-%s", tenantId, podName)
			ctx["podnamespace"] = DefaultBenchNamespace
		}

		podYaml, err := fillOutTemplate(defaultPodTemp, ctx)
		if err != nil {
			log.Printf("fail to submit pods(%s) by tenant(%d): %s", podName, tenantId, err)
			return err
		}

		obj, err := yamlBytsToObject(scheme.Scheme, podYaml)
		if err != nil {
			log.Printf("fail to submit pods(%s) by tenant(%d): %s", podName, tenantId, err)
			return err
		}

		pod, ok := obj.(*v1.Pod)
		if !ok {
			log.Printf("fail to assert pod(%s) by tenant(%d)", podName, tenantId)
			return err
		}
		if err = cli.Create(context.TODO(), pod); err != nil {
			log.Printf("fail to submit pod(%s) by tenant(%d): %s", podName, tenantId, err)
			return err
		}
		log.Printf("pod(%s) submitted by tenant %d", podName, tenantId)
		<-time.After(time.Duration(bbe.PodInterval) * time.Second)
	}
	return nil
}

func (bbe *BaseBenchExecutor) RunBaseBench() error {

	// submit pods
	waitingPods := (bbe.NumPod / bbe.NumTenants) * bbe.NumTenants
	podPerTenant := bbe.NumPod / bbe.NumTenants

	if bbe.ShareNamespace {
		sharedNs := DefaultBenchNamespace
		if err := bbe.CliLst[0].Create(context.TODO(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: sharedNs,
			},
		}); err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	for i := range bbe.CliLst {
		wg.Add(1)
		go bbe.SubmitPods(bbe.CliLst[i], podPerTenant, i, &wg)
	}
	wg.Wait()
	log.Printf("all pods submitted")

	// periodically check pod status
CHECKPOD:
	for waitingPods != 0 {
		log.Printf("checking waiting pods, there are %d pod remaining", waitingPods)
		<-time.After(20 * time.Second)
		for i := 0; i < bbe.NumTenants; i++ {
			tnNs := DefaultBenchNamespace
			if !bbe.ShareNamespace {
				tnNs = fmt.Sprintf("%s-%d", DefaultBenchNamespace, i)
			}
			log.Printf("check pod status in namespace(%s)", tnNs)
			pl := &v1.PodList{}
			if err := bbe.CliLst[0].List(context.TODO(), pl, client.InNamespace(tnNs)); err != nil {
				log.Printf("fail to list pod: %s", err)
			}
			for _, p := range pl.Items {
				log.Printf("checking pod(%s)", p.GetName())
				tmpPodName := p.GetName()
				if !bbe.ShareNamespace {
					tmpPodName = fmt.Sprintf("%s-%s", tnNs, p.GetName())
				}
				if p.Status.Phase == v1.PodRunning {
					if _, exist := bbe.RuntimeStatics[tmpPodName]; exist {
						continue
					}
					log.Printf("new %s/pod(%s) finished", tnNs, p.GetName())
					bbe.RuntimeStatics[tmpPodName] = &BasePodStatiscs{
						CreationTimestamp: p.GetCreationTimestamp().Unix(),
						ReadyTimestamp:    p.Status.Conditions[0].LastTransitionTime.Unix(),
					}
					waitingPods--
					if waitingPods == 0 {
						break CHECKPOD
					}
				}
			}
		}
	}
	return nil
}
