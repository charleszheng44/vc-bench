package kube

import (
	"context"
	"errors"
	"fmt"
	"log"

	yaml "gopkg.in/yaml.v2"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenancyv1alpha1 "github.com/kubernetes-sigs/multi-tenancy/incubator/virtualcluster/pkg/apis/tenancy/v1alpha1"
	"github.com/kubernetes-sigs/multi-tenancy/incubator/virtualcluster/pkg/syncer/conversion"
)

func UpdateKubeConfig(admKbCfgByts []byte, newServerUrl string) ([]byte, error) {
	admKbCfgMap := make(map[string]interface{})
	if err := yaml.Unmarshal(admKbCfgByts, &admKbCfgMap); err != nil {
		return nil, err
	}
	for k := range admKbCfgMap {
		if k == "clusters" {
			// locate the list of clusters
			clsLst := admKbCfgMap[k].([]interface{})
			// NOTE by default, admin-kubeconfig contains only
			// one cluster
			clsInfo := clsLst[0].(map[interface{}]interface{})
			clsMap := clsInfo["cluster"].(map[interface{}]interface{})
			for k2 := range clsMap {
				if k2 == "server" {
					// locate the server field
					clsMap[k2] = newServerUrl
				}
			}
			delete(clsMap, "certificate-authority-data")
			clsMap["insecure-skip-tls-verify"] = true
			// assign it back
			clsInfo["cluster"] = clsMap
			clsLst[0] = clsInfo
			admKbCfgMap[k] = clsLst
		}
	}
	// marshal map to bytes
	newAdmKbCfgByts, err := yaml.Marshal(admKbCfgMap)
	if err != nil {
		return nil, err
	}
	return newAdmKbCfgByts, nil
}

func GetNodePortUrl(cli client.Reader, vc *tenancyv1alpha1.Virtualcluster) (addr string, err error) {
	// 1. get node ip
	nodeIP, err := getNodeIP(cli)
	if err != nil {
		return
	}
	// 2. get node port
	apisvcName, err := getApiserverSvcName(cli, vc)
	if err != nil {
		return
	}
	port, err := getNodePort(cli, apisvcName, conversion.ToClusterKey2(vc))
	if err != nil {
		return
	}
	return fmt.Sprintf("https://%s:%d", nodeIP, port), nil
}

func getNodeIP(cli client.Reader) (string, error) {
	nodeList := &v1.NodeList{}
	if err := cli.List(context.TODO(), nodeList); err != nil {
		return "", err
	}

	if len(nodeList.Items) == 0 {
		return "", errors.New("there is no available node")
	}
	var (
		nodeIP          string
		foundInternalIP bool
	)
	for _, addr := range nodeList.Items[0].Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			nodeIP = addr.Address
			foundInternalIP = true
			break
		}
	}
	if !foundInternalIP {
		return "", errors.New("can't find the internal IP")
	}
	log.Printf("the nodeip is %s", nodeIP)
	return nodeIP, nil
}

func getNodePort(cli client.Reader, svcName, namespace string) (port int32, err error) {
	svc := &v1.Service{}
	if err = cli.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      svcName,
	}, svc); err != nil && apierrors.IsNotFound(err) {
		return
	}
	err = nil
	if len(svc.Spec.Ports) == 0 {
		err = fmt.Errorf("%s doesn't contain port", svcName)
		return
	}
	port = svc.Spec.Ports[0].NodePort
	log.Printf("the port is %d", port)
	return
}

func getApiserverSvcName(cli client.Reader, vc *tenancyv1alpha1.Virtualcluster) (apisvcName string, err error) {
	cvName := vc.Spec.ClusterVersionName
	if cvName == "" {
		err = fmt.Errorf("Virtualcluster(%s) spec doesn't contain ClusterVersionName field", vc.GetName())
		return
	}
	cv := &tenancyv1alpha1.ClusterVersion{}
	err = cli.Get(context.TODO(), types.NamespacedName{
		Namespace: "default",
		Name:      cvName,
	}, cv)
	if err != nil && !apierrors.IsNotFound(err) {
		return
	}
	err = nil

	apisvcName = cv.Spec.APIServer.Service.GetName()
	if apisvcName == "" {
		err = fmt.Errorf("Service name of APIServer is not found for Virtualcluster(%s)", vc.GetName())
		return
	}
	return
}
