package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CatchZeng/dingtalk/pkg/dingtalk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	dingtalkClient *dingtalk.Client
)

func main() {

	var kubeconfig *string
	var url *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	url = flag.String("url", "", "metrics url ")

	flag.Parse()

	if *url == "" {
		log.Fatalln("set metrics url with --url")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("use kubeconfig with failed |%s| and try to run with inCluster\n", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	token := os.Getenv("TOKEN")
	secret := os.Getenv("SECRET")
	if token != "" && secret != "" {
		dingtalkClient = dingtalk.NewClient(token, secret)
	}

	result, code := get(context.TODO(), *url)
	if code == 200 {
		datamap := parseMetricsForPod(result)
		// log.Println("datamap:", datamap)
		for ns, mpods := range datamap {
			if len(mpods) > 0 {
				log.Printf("select pod from namespace:%s\n", ns)
				pods, err := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					panic(err)
				}

				apidataMap := make(map[string]int)
				for i := 0; i < len(pods.Items); i++ {
					apidataMap[pods.Items[i].Name] = 0
				}

				for i := 0; i < len(mpods); i++ {
					if _, ok := apidataMap[mpods[i]]; !ok {
						log.Printf("this pod exist in ksm and not exist k8s cluster:%s\n", mpods[i])

						if dingtalk!=nil{
							content := fmt.Sprintf("======KSM-Checker======,\n\n stale pod:%s", mpods[i])
							msg := dingtalk.NewMarkdownMessage().SetMarkdown("KSM-Checker", content).SetAt([]string{""}, false)
							dingtalkClient.Send(msg)
						}
						
					}
				}

				// log.Println("pods.pods:", len(pods.Items))
			}
		}
	} else {
		log.Fatalf("get metrics from %s failed!", url)
	}
}

func parseMetricsForPod(metrics string) map[string][]string {
	resutMap := make(map[string][]string)
	//key namespace  value: []string(pod)
	strs := strings.Split(metrics, "\n")
	for i := 0; i < len(strs); i++ {
		str := strs[i]
		if !strings.Contains(str, "#") {
			if strings.Contains(str, "kube_pod_info") {
				s1 := strings.Split(str, "{")[1]
				s2 := strings.Split(s1, "}")[0]

				keyvalues := strings.Split(s2, ",")
				namespace := ""
				pod := ""
				for j := 0; j < len(keyvalues); j++ {
					kv := strings.Split(keyvalues[j], "=")
					if kv[0] == "namespace" {
						namespace = strings.ReplaceAll(kv[1], "\"", "")
					}
					if kv[0] == "pod" {
						pod = strings.ReplaceAll(kv[1], "\"", "")
					}
					if namespace != "" && pod != "" {
						break
					}
					// log.Printf("key:%s,value:%s\n", kv[0], kv[1])
				}
				if _, ok := resutMap[namespace]; !ok {
					var tmp []string
					resutMap[namespace] = tmp
				}
				resutMap[namespace] = append(resutMap[namespace], pod)
			}
		}
	}
	return resutMap
}

func get(context context.Context, url string) (string, int) {

	var resp *http.Response
	var err error

	client := &http.Client{Timeout: time.Duration(3) * time.Second}
	log.Println("begin.req:", url)
	resp, err = client.Get(url)

	if err != nil {
		log.Println("req.failed!", err)
		return "", resp.StatusCode
	}

	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			log.Println("write.resp.content.failed!", resp.StatusCode, err)
			return "", 500
		}
	}

	return result.String(), resp.StatusCode
}
