package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/charleszheng44/vc-bench/pkg/vcbench"
)

var (
	tenantsKbCfgPath       string
	outDataDir             string
	syncerAddr             string
	kubeletAddr            string
	scrapeInterval         int
	scrapeKubeletInterval  int
	numPod                 int
	numOfVC                int
	tenantInterval         int
	podInterval            int
	syncerStandaloneMinute int

	targetNs         string
	tenantRangeStart int
	tenantRangeEnd   int

	kubeconfigPathBase string
	numeTenants        int
	numePodBase        int
	podIntervalBase    int
	shareNs            bool

	runBenchFlagSet     *flag.FlagSet
	runBaseBenchFlagSet *flag.FlagSet
	cleanupFlagSet      *flag.FlagSet
)

const TimeOutputFmt = "20101010150405"

func init() {
	// set default tenantsKbCfgPath
	defaultTenantKbCfgPath := os.Getenv("KUBECONFIG")
	if defaultTenantKbCfgPath == "" {
		defaultTenantKbCfgPath = path.Join(os.Getenv("HOME"), ".kube", "config")
	}

	runBaseBenchFlagSet = flag.NewFlagSet("base", flag.ExitOnError)
	runBaseBenchFlagSet.StringVar(&kubeconfigPathBase, "kubeconfig", "", "The path to the kubeconfig file")
	runBaseBenchFlagSet.IntVar(&numePodBase, "numPod", 1, "number of pods to be submitted")
	runBaseBenchFlagSet.IntVar(&numeTenants, "numTenants", 1, "number of pods to be submitted")
	runBaseBenchFlagSet.IntVar(&podIntervalBase, "podInterval", 0, "pod submission interval")
	runBaseBenchFlagSet.BoolVar(&shareNs, "shareNs", false, "if use a shared benchmark namespace")

	// command options for subcommand "run"
	runBenchFlagSet = flag.NewFlagSet("run", flag.ExitOnError)
	runBenchFlagSet.StringVar(&tenantsKbCfgPath, "tenantkbcfg", defaultTenantKbCfgPath, "The kubeconfig file of the k8s that holds tenant masters ")
	runBenchFlagSet.StringVar(&outDataDir, "outDataDir", "", "The path to the directory that will store benchmark data")
	runBenchFlagSet.StringVar(&syncerAddr, "syncerAddr", "", "The address of the syncer pod")
	runBenchFlagSet.StringVar(&kubeletAddr, "kubeletAddr", "", "The address of the kubelet")
	runBenchFlagSet.IntVar(&numPod, "numpod", 10, "Total amount of pods that will be created")
	runBenchFlagSet.IntVar(&scrapeInterval, "scrapeInterval", 20, "The interval for scraping metrics from syncer pod")
	runBenchFlagSet.IntVar(&scrapeKubeletInterval, "scrapeKubeletInterval", 30, "The interval for scraping metrics from kubelet")
	runBenchFlagSet.IntVar(&tenantInterval, "tntintvl", 0, "The submission interval among tenants")
	runBenchFlagSet.IntVar(&podInterval, "podintvl", 0, "The submission interval of pods in one tenant")
	runBenchFlagSet.IntVar(&numOfVC, "numvc", 10, "Number of Virtualcluster that will be used")
	runBenchFlagSet.IntVar(&syncerStandaloneMinute, "syncer-alone-minutes", 5, "Number of minutes for syncer to standalone after podbench successfully completing")

	// command options for subcommand "clean"
	cleanupFlagSet = flag.NewFlagSet("clean", flag.ExitOnError)
	cleanupFlagSet.StringVar(&targetNs, "targetNs", vcbench.DefaultBenchNamespace, "")
}

func main() {

	if len(os.Args) <= 1 {
		log.Fatal("please specify a subcommand: 'run', 'clean', or 'base'")
		os.Exit(1)
	}

	switch os.Args[1] {

	case "base":
		runBaseBenchFlagSet.Parse(os.Args[2:])
		baseOutDataDir := fmt.Sprintf("base-pod%d-tenants%d-podsleep%d-shareNs-%v", numePodBase, numeTenants, podIntervalBase, shareNs)
		if err := os.MkdirAll(baseOutDataDir, os.ModePerm); err != nil {
			log.Fatalf("fail to run base benchmark: %s", err)
		}
		baseLogFile := path.Join(baseOutDataDir,
			fmt.Sprintf("base-pod%d-tenants%d-podsleep%d.data", numePodBase, numeTenants, podIntervalBase))
		baseDiffLogFile := path.Join(baseOutDataDir,
			fmt.Sprintf("base-pod%d-tenants%d-podsleep%d.diff", numePodBase, numeTenants, podIntervalBase))
		baselogFd, err := os.OpenFile(baseLogFile, os.O_CREATE|os.O_RDWR, 0644)
		defer baselogFd.Close()
		if err != nil {
			log.Fatalf("fail to run base benchmark: %s", err)
		}
		baseDiffLogFd, err := os.OpenFile(baseDiffLogFile, os.O_CREATE|os.O_RDWR, 0644)
		defer baseDiffLogFd.Close()
		if err != nil {
			log.Fatalf("fail to run base benchmark: %s", err)
		}

		bbe, err := vcbench.NewBaseBenchExecutor(kubeconfigPathBase, numePodBase, podIntervalBase, numeTenants, shareNs)
		if err != nil {
			log.Fatalf("fail to generate bench executor: %s", err)
		}
		err = bbe.RunBaseBench()
		if err != nil {
			log.Fatalf("fail to run base benchmark: %s", err)
		}
		baselogFd.WriteString("#podName,creationTimestamp,readyTimestamp\n")
		baseDiffLogFd.WriteString("#podName,latency\n")
		for pn, pi := range bbe.RuntimeStatics {
			baselogFd.WriteString(fmt.Sprintf("%s,%d,%d\n", pn, pi.CreationTimestamp, pi.ReadyTimestamp))
			baseDiffLogFd.WriteString(fmt.Sprintf("%s,%d\n", pn, pi.ReadyTimestamp-pi.CreationTimestamp))
		}

	case "run":
		runBenchFlagSet.Parse(os.Args[2:])
		// 1. create directory for storing benchmark results
		if outDataDir == "" {
			outDataDir = fmt.Sprintf("pod%d-vc%d-vcsleep%d-podsleep%d-%s",
				numPod, numOfVC,
				tenantInterval, podInterval,
				time.Now().Format(TimeOutputFmt))
		}
		if err := os.MkdirAll(outDataDir, os.ModePerm); err != nil {
			log.Fatalf("fail to create output data directory(outDataDir): %s", err)
		}
		outDataPath := path.Join(outDataDir, fmt.Sprintf("%s.log", outDataDir))
		outDataPathDiff := path.Join(outDataDir, fmt.Sprintf("%s.diff", outDataDir))

		logFd, err := os.OpenFile(outDataPath, os.O_CREATE|os.O_RDWR, 0644)
		defer logFd.Close()
		if err != nil {
			log.Fatalf("fail to open file %s: %s", outDataPath, err)
		}
		logDiffFd, err := os.OpenFile(outDataPathDiff, os.O_CREATE|os.O_RDWR, 0644)
		defer logDiffFd.Close()
		if err != nil {
			log.Fatalf("fail to open file %s: %s", outDataPathDiff, err)
		}

		// 2. run benchmark
		be, err := vcbench.NewBenchExecutor(tenantsKbCfgPath, numPod, tenantInterval, podInterval, numOfVC)
		if err != nil {
			log.Fatalf("fail to initialize bench executor: %s", err)
		}
		stop := make(chan struct{})

		go vcbench.ScrapeSyncer(stop, outDataDir, syncerAddr, scrapeInterval)
		go vcbench.ScrapeKubelet(stop, outDataDir, kubeletAddr, scrapeKubeletInterval)

		err = be.RunBench()
		if err != nil {
			log.Fatalf("fail to run bench: %s", err)
		}
		// log.Printf("benchmark successfully complete, will wait for %d minutes", syncerStandaloneMinute)
		// <-time.After(time.Duration(syncerStandaloneMinute) * time.Minute)
		close(stop)

		log.Printf("writing runtime data to log(%s)", outDataPath)
		logFd.WriteString("#podName,tenantCreation,dwsDequeue,superCreation,superReady,uwsDequeue,tenantUpdate\n")
		logDiffFd.WriteString("#podName,dwsQDelay,dwsProcessDelay,superCreationTime,uwsQDelay,tenantUpdateTime,total\n")
		for pn, rs := range be.RuntimeStatics {
			logFd.WriteString(fmt.Sprintf("%s,%d,%d,%d,%d,%d,%d\n", pn,
				rs.TenantCreation,
				rs.DwsDequeue,
				rs.SuperCreation,
				rs.SuperReady,
				rs.UwsDequeue,
				rs.SuperUpdate))

			logDiffFd.WriteString(fmt.Sprintf("%s,%d,%d,%d,%d,%d,%d\n", pn,
				rs.DwsDequeue-rs.TenantCreation,
				rs.SuperCreation-rs.DwsDequeue,
				rs.SuperReady-rs.SuperCreation,
				rs.UwsDequeue-rs.SuperReady,
				rs.SuperUpdate-rs.UwsDequeue,
				rs.SuperUpdate-rs.TenantCreation))
		}

	case "clean":
		cleanupFlagSet.Parse(os.Args[2:])
		be, err := vcbench.NewBenchExecutor(tenantsKbCfgPath, 0, 0, 0, int(^uint(0)>>1))
		if err != nil {
			log.Fatalf("fail to initialize bench executor: %s", err)
		}
		log.Println("will try to remove all benchmark namespace")
		be.CleanUp(targetNs)

	default:
		log.Fatalf("unsupport action: %s", os.Args[1])
		os.Exit(1)
	}
}
