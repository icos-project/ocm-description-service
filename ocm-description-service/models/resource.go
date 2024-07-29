/*
  OCM-DESCRIPTION-SERVICE
  Copyright © 2022-2024 EVIDEN

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

  This work has received funding from the European Union's HORIZON research
  and innovation programme under grant agreement No. 101070177.
*/

package models

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	workv1 "open-cluster-management.io/api/work/v1"
)

// type Resource struct {
// 	ID           string             `json:"resource_uuid"`
// 	JobID        string             `json:"job_id" validate:"omitempty,uuid4"`
// 	ManifestName string             `json:"resource_name"`
// 	NodeTarget   string             `json:"node_target"`
// 	Conditions   []metav1.Condition `json:"conditions,omitempty"`
// 	UpdatedAt    time.Time          `json:"updatedAt"`
// }

// type Status struct {
// 	Conditions []metav1.Condition `json:"conditions,omitempty"`
// }

func CreateManifestWork(target Target, manifestWorkYaml string) (string, error) {
	name := "deploy-test-"
	namespace := target.ClusterName
	// TODO validate if work doesnt exist already
	if ExistsManifestWork(namespace, name) {
		fmt.Println("ManifestWork " + name + " already exists!")
		return "", errors.New("ManifestWork " + name + " already exists!") //message
	}
	// fmt.Println(work.Spec.Workload.Manifests)

	var manifestWork *workv1.ManifestWork
	// decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
	decoder := scheme.Codecs.UniversalDeserializer()
	manifestWork = &workv1.ManifestWork{}
	err := runtime.DecodeInto(decoder, []byte(manifestWorkYaml), manifestWork)
	if err != nil {
		fmt.Println(err)
		// panic(err)
	}
	fmt.Println("Sending manifest to OCM...")
	manifestWork, err = clientsetWorkOper.WorkV1().ManifestWorks(target.ClusterName).
		Create(context.TODO(), manifestWork, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("ERROR: ", err)
		// panic(err)
		return "", errors.New(err.Error())
	}
	return string(manifestWork.GetUID()), err
}

func GetManifestWork(namespace string, manifestWorkName string) (*workv1.ManifestWork, error) {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	manifest, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).
		Get(context.TODO(), manifestWorkName, metav1.GetOptions{})
	// log.Debug("ExistsManifestWork: " + manifestWorkName + " " + strconv.FormatBool(err == nil)) //err.Error())
	if err == nil {
		fmt.Println("ERROR: ", err)
	}
	return manifest, err
}

func CheckStatusManifestWork(namespace string, manifestWorkName string) *workv1.ManifestWorkStatus {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	// log.Debug("Obtaining status... ") //err.Error())
	result, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).
		Get(context.TODO(), manifestWorkName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Error obtaining ManifestWork status")
	}
	//	setManifestWorkCache(namespace, manifestWorkName)
	//}
	return &result.Status // TODO update to be an array
}

func ExistsManifestWork(namespace string, manifestWorkName string) bool {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	_, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).Get(context.TODO(), manifestWorkName, metav1.GetOptions{})
	// log.Debug("ExistsManifestWork: " + manifestWorkName + " " + strconv.FormatBool(err == nil)) //err.Error())
	//if err == nil {
	//	setManifestWorkCache(namespace, manifestWorkName)
	//}
	return err == nil
}

func DeleteManifestWork(namespace string, manifestWorkName string) bool {
	if !ExistsManifestWork(namespace, manifestWorkName) {
		fmt.Println("Manifest " + manifestWorkName + " does not exist!")
		return false
	}
	//err := clientset.CoreV1().Services(namespace).Delete(context.TODO(), serviceName, metav1.DeleteOptions{})
	err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).Delete(context.TODO(), manifestWorkName, metav1.DeleteOptions{})
	fmt.Println("DeleteManifestWork: " + manifestWorkName + " " + strconv.FormatBool(err == nil))
	return err == nil
}

func ListManifestWork(namespace string) *workv1.ManifestWorkList {
	manifestlist, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error obtaining ManifestWorkList")
	}
	return manifestlist
}

func ResourceSync() ([]Resource, error) {
	var err error
	var resources []Resource
	// get all manifestworks
	var manifestStatus *workv1.ManifestWorkStatus
	// var managedClusters clusterv1.ManagedClusterList
	managedClusters, err := clientsetClusterOper.ClusterV1().ManagedClusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error obtaining managed clusters")
	}
	for _, managedCluster := range managedClusters.Items {
		allManifestWorks := ListManifestWork(managedCluster.Name)
		// for each manifestwork
		if len(allManifestWorks.Items) > 0 {
			for _, manifestWork := range allManifestWorks.Items {
				// get status
				manifestStatus = &manifestWork.Status
				// find job with the corresponding UID, should I assume it exists?
				// manifestUID := uuid.MustParse(string(string(manifestWork.UID)))
				resource := Resource{
					ResourceUUID: string(manifestWork.UID),
					ResourceName: manifestWork.Name,
					Conditions:   manifestStatus.Conditions,
				}
				resources = append(resources, resource)
			}
		} else {
			fmt.Println("No Resources were found during sync up process for cluster: " + managedCluster.Name)
		}
	}
	return resources, err
}

func PatchManifestWork(namespace string, manifestWorkName string, manifestWork workv1.ManifestWork) bool {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})

	_, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).Update(context.TODO(), &manifestWork, metav1.UpdateOptions{})
	// log.Debug("ExistsManifestWork: " + manifestWorkName + " " + strconv.FormatBool(err == nil)) //err.Error())
	//if err == nil {
	//	setManifestWorkCache(namespace, manifestWorkName)
	//}
	return err == nil
}
