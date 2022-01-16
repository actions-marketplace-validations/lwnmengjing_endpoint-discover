/*
Copyright 2016 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/sanity-io/litter"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	serviceList, err := clientset.CoreV1().Services("beta").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("There are %d services in the cluster\n", len(serviceList.Items))

	cm := &v1.ConfigMap{}
	cm.Name = "grpc-endpoints"
	cm.Namespace = "beta"
	endpoints := make([]Endpoint, 0)
	for i := range serviceList.Items {
		for j := range serviceList.Items[i].Spec.Ports {
			if strings.Index(serviceList.Items[i].Spec.Ports[j].Name, "grpc") > -1 {
				var port int
				switch serviceList.Items[i].Spec.Ports[j].TargetPort.String() {
				case "http":
					port = 80
				case "https":
					port = 443
				default:
					port = serviceList.Items[i].Spec.Ports[j].TargetPort.IntValue()
				}
				endpoints = append(endpoints, Endpoint{
					Name:     serviceList.Items[i].Name,
					Endpoint: fmt.Sprintf("%s.%s:%d", serviceList.Items[i].Name, "beta", port),
				})
			}
		}
	}
	out, err := yaml.Marshal(endpoints)
	if err != nil {
		panic(err)
	}
	cm.Data = map[string]string{
		"endpoints.yaml": string(out),
	}

	_, err = clientset.CoreV1().ConfigMaps("beta").Get(context.TODO(), "grpc-endpoints", metav1.GetOptions{})

	if errors.IsNotFound(err) {
		cm, err = clientset.CoreV1().ConfigMaps("beta").Create(context.TODO(), cm, metav1.CreateOptions{})
	} else {
		cm, err = clientset.CoreV1().ConfigMaps("beta").Update(context.TODO(), cm, metav1.UpdateOptions{})
	}
	//查询是否存在
	if err != nil {
		panic(err)
	}
	litter.Dump(cm)
}

type Endpoint struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
}