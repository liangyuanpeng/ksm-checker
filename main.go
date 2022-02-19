package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	result, code := get(context.TODO(), "http://localhost:8080/metrics")
	if code == 200 {
		datamap := parseMetricsForPod(result)
		log.Println("datamap:", datamap)
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
						namespace = kv[1]
					}
					if kv[0] == "pod" {
						pod = kv[1]
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
