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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"icos/server/ocm-description-service/utils/logs"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clustermanager "open-cluster-management.io/api/client/operator/clientset/versioned/typed/operator/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	yamlEncode "sigs.k8s.io/yaml"

	workclient "open-cluster-management.io/api/client/work/clientset/versioned"
)

var (
	jobmanagerBaseURL    = os.Getenv("JOBMANAGER_URL")
	clientset            *kubernetes.Clientset
	clientsetWorkOper    workclient.Interface
	clientsetClusterOper clusterclient.Interface
	clientOperator       *clustermanager.OperatorV1Client
	JobTypeToString      = map[JobType]string{
		CreateDeployment:  "CreateDeployment",
		UpdateDeployment:  "UpdateDeployment",
		DeleteDeployment:  "DeleteDeployment",
		ReplaceDeployment: "ReplaceDeployment",
	}
)

// Job struct with fields ordered by size to reduce memory padding.
// Represents a job with various attributes
type Metadata struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Base entities with UUID and UINT
type BaseUUID struct {
	Metadata
	ID string `json:"id"`
}

type BaseUINT struct {
	Metadata
	ID uint32 `json:"id"`
}

type Job struct {
	BaseUUID
	JobGroupID string `json:"job_group_id"`
	//ResourceUID         string           `json:"uuid,omitempty"`
	OwnerID             string          `json:"owner_id,omitempty"`
	JobGroupName        string          `json:"job_group_name"`
	JobGroupDescription string          `json:"job_group_description,omitempty"`
	Type                JobType         `json:"type,omitempty"`
	SubType             RemediationType `json:"sub_type,omitempty"`
	State               JobState        `json:"state,omitempty"`
	Manifests           []PlainManifest `json:"manifests"`
	Target              Target          `json:"targets,omitempty"`
	//Locker              *bool            `json:"locker,omitempty"`
	Orchestrator OrchestratorType `json:"orchestrator"`
	Resource     *Resource        `json:"resource,omitempty"`
	Namespace    string           `json:"namespace,omitempty"`
}

type JobState int
type ResourceState string
type ConditionStatus string

type Resource struct {
	BaseUUID
	JobID        string             `json:"job_id"`
	ResourceUUID string             `json:"resource_uuid,omitempty"`
	ResourceName string             `json:"resource_name,omitempty"`
	Conditions   []metav1.Condition `json:"conditions,omitempty"`
}

type PlainManifest struct {
	BaseUINT
	JobID      string `json:"-"`
	YamlString string `json:"yamlString"`
}

type Target struct {
	BaseUINT
	JobID        string           `json:"-"`
	ClusterName  string           `json:"cluster_name"`
	NodeName     string           `json:"node_name,omitempty"`
	Orchestrator OrchestratorType `json:"orchestrator"`
}

// Type declarations
type (
	State            int
	JobType          int
	OrchestratorType string
	RemediationType  string
)

// Constants for OrchestratorType and RemediationType
const (
	OCM          OrchestratorType = "ocm"
	NUVLA        OrchestratorType = "nuvla"
	ScaleUp      RemediationType  = "scale-up"
	ScaleDown    RemediationType  = "scale-down"
	ScaleOut     RemediationType  = "scale-out"
	ScaleIn      RemediationType  = "scale-in"
	Reallocation RemediationType  = "reallocation"
)

// Constants for State and JobType
const (
	// Valid condition types are:
	// 1. Applied represents workload in ManifestWork is applied successfully on managed cluster.
	// 2. Progressing represents workload in ManifestWork is being applied on managed cluster.
	// 3. Available represents workload in ManifestWork exists on the managed cluster.
	// 4. Degraded represents the current state of workload does not match the desired
	Applied JobState = iota + 1
	Progressing
	Available
	Degraded

	CreateDeployment JobType = iota + 1
	DeleteDeployment
	UpdateDeployment
	ReplaceDeployment
)

// Configuration and Initialization
// ------------------------------------------------)

// InClusterConfig sets up Kubernetes client configurations for in-cluster and out-of-cluster environments.
func InClusterConfig() error {
	config, err := rest.InClusterConfig()

	// Outside of the cluster for development
	if err != nil {
		//panic(err.Error())
		var kubeconfig *string
		logs.Logger.Println("The home folder is: ", homedir.HomeDir())
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", "/home/mgallardo/.kube/config", "absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clientsetWorkOper, err = workclient.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clientsetClusterOper, err = clusterclient.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clientOperator, err = clustermanager.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return err
}

// Job Execution and Management
// ------------------------------------------------

// Execute executes the job based on its type, such as creating, updating, or deleting a deployment.
func Execute(j *Job) (*Job, error) {
	jobType := getJobTypeString(j.Type)
	logs.Logger.Println("Executing job type:", jobType)

	switch j.Type {
	case CreateDeployment:
		return createDeployment(j)
	case UpdateDeployment:
		return updateDeployment(j)
	case DeleteDeployment:
		return deleteDeployment(j)
	case ReplaceDeployment:
		return replaceDeployment(j)
	default:
		err := fmt.Errorf("job type not supported: %s", jobType)
		logs.Logger.Println(err)
		return nil, err
	}
}

// createDeployment creates a new deployment for the given job and updates the job's resource details.
func createDeployment(j *Job) (*Job, error) {
	return createAndApplyManifestWork(j)
}

// Helper function to create and apply ManifestWork, then update the job resource details
func createAndApplyManifestWork(j *Job) (*Job, error) {
	logs.Logger.Println("Creating Work for Job:", j.ID)

	// Create the ManifestWork
	mw, err := createManifestWork(j)
	if err != nil {
		logErrorAndSetJobState("Error creating ManifestWork", j, Degraded)
		return nil, err
	}

	// Extract namespace and resource UUID
	namespace := mw.Namespace
	resUUID := string(mw.GetUID())

	if resUUID != "" {
		logs.Logger.Println("ManifestWork UID: ", resUUID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		appliedManifestWork, err := waitForAppliedManifestWork(namespace, mw.Name, ctx)
		if err != nil {
			logs.Logger.Println("Error obtaining applied ManifestWork status:", err)
			j.State = Degraded
			return nil, err
		}

		// Update job with new resource details
		//j.ResourceUID = resUUID
		j.UpdateJobResource(appliedManifestWork)
	}

	return j, nil
}

// bluegreen deployment strategy
func replaceDeployment(j *Job) (*Job, error) {
	logs.Logger.Println("Replacing Work for Job:", j.ID)

	// Fetch old deployment
	oldManifestWork, err := fetchManifestWork(j.Target.ClusterName, j.Resource.ResourceName, nil)
	if err != nil {
		logErrorAndSetJobState("Error obtaining applied ManifestWork status", j, Degraded)
		return nil, err
	}

	updatedManifests := []workv1.Manifest{}

	for _, stringManifest := range j.Manifests {
		obj, err := decodeYAMLToObject(stringManifest.YamlString)
		if err != nil {
			logs.Logger.Println("Error unmarshaling manifest:", err)
			continue
		}
		updateNamespaceAndAnnotations(obj, j.Namespace, j.JobGroupName, j.Resource.ResourceName, j.JobGroupID, j.Resource.ID)
		rawExtension := runtime.RawExtension{Object: obj}
		manifest := workv1.Manifest{RawExtension: rawExtension}
		// usar para el replace deployment
		updatedManifests = append(updatedManifests, manifest)
	}

	oldManifestWork.Spec.Workload.Manifests = updatedManifests

	updatedManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Update(context.TODO(), oldManifestWork, metav1.UpdateOptions{})

	if err != nil {
		logErrorAndSetJobState("Error updating ManifestWork", j, Degraded)
		return nil, err
	}

	j.UpdateJobResource(updatedManifestWork)

	return j, nil
}

// Used in remediation actions
// updateDeployment updates an existing deployment for the given job and updates the job's resource details.
func updateDeployment(j *Job) (*Job, error) {
	logs.Logger.Println("Updating work for Job:", j.ID)
	switch j.SubType {

	case ScaleUp, ScaleDown, ScaleOut, ScaleIn:
		return updateDeploymentAttributes(j)
	case Reallocation:
		return deleteDeployment(j)
	default:
		logErrorAndSetJobState("Job Sub Type does not exist", j, Degraded)
		return nil, fmt.Errorf("job sub type does not exist: %v", j.SubType)
	}
}

// deleteDeployment deletes the deployment associated with the given job and clears the job's resource details.
func deleteDeployment(j *Job) (*Job, error) {
	logs.Logger.Println("Deleting deployment for Job:", j.ID)

	err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Delete(context.TODO(), j.Resource.ResourceName, metav1.DeleteOptions{})
	if err != nil {
		logErrorAndSetJobState("Error obtaining applied ManifestWork status", j, Degraded)
		return nil, err
	}

	logs.Logger.Printf("Successfully deleted deployment for Job: %s\n", j.ID)
	j.State = Applied
	return j, nil
}

// OCM Manifest Work Operations
// ------------------------------------------------
// createManifestWork creates a manifest work for the given job in the specified cluster.
func createManifestWork(j *Job) (*workv1.ManifestWork, error) {
	manifestWork := GenerateManifestWork(j)
	createdManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Create(context.TODO(), manifestWork, metav1.CreateOptions{})
	if err != nil {
		logErrorAndSetJobState("Error creating ManifestWork", j, Degraded)
		return nil, fmt.Errorf("error creating ManifestWork: %v", err)
	}
	return createdManifestWork, nil
}

// fetchManifestWork retrieves the manifest work object from the specified namespace and name.
func fetchManifestWork(namespace, manifestWorkName string, ctx context.Context) (*workv1.ManifestWork, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	manifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).Get(ctx, manifestWorkName, metav1.GetOptions{})
	if err != nil || manifestWork == nil {
		return nil, fmt.Errorf("error obtaining applied ManifestWork status: %v", err)
	}
	return manifestWork, nil
}

// waitForAppliedManifestWork polls for the applied ManifestWork until it is found or the context times out.
func waitForAppliedManifestWork(namespace, name string, ctx context.Context) (*workv1.ManifestWork, error) {
	var appliedManifestWork *workv1.ManifestWork
	var err error

	for {
		appliedManifestWork, err = fetchManifestWork(namespace, name, ctx)
		if err != nil {
			logs.Logger.Println("Error obtaining applied ManifestWork status:", err)
		}

		if appliedManifestWork != nil && len(appliedManifestWork.Status.Conditions) > 0 {
			break
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context timed out while waiting for applied ManifestWork status")
		case <-time.After(500 * time.Millisecond): // Poll interval
		}
	}

	return appliedManifestWork, nil
}

// GenerateManifestWork generates a manifest work object for the given job.
func GenerateManifestWork(j *Job) *workv1.ManifestWork {
	work := workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.Resource.ResourceName + "-",
			Namespace:    j.Target.ClusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{},
		},
	}

	namespaceManifest, err := generateNamespaceManifest(j.Namespace)
	if err != nil {
		logs.Logger.Println("Error generating namespace manifest:", err)
	}
	work.Spec.Workload.Manifests = append(work.Spec.Workload.Manifests, namespaceManifest)

	for _, stringManifest := range j.Manifests {
		obj, err := decodeYAMLToObject(stringManifest.YamlString)
		if err != nil {
			logs.Logger.Println("Error unmarshaling manifest:", err)
			continue
		}
		updateNamespaceAndAnnotations(obj, j.Namespace, j.JobGroupName, j.Resource.ResourceName, j.JobGroupID, j.Resource.ID)
		rawExtension := runtime.RawExtension{Object: obj}
		manifest := workv1.Manifest{RawExtension: rawExtension}
		logs.Logger.Print("------Inside GenerateManifestWork----------")
		logs.Logger.Print("Manifest Kind: ", manifest.RawExtension.Object.GetObjectKind().GroupVersionKind().Kind)
		logs.Logger.Print("------Inside GenerateManifestWork----------")
		// usar para el replace deployment
		work.Spec.Workload.Manifests = append(work.Spec.Workload.Manifests, manifest)
	}
	return &work
}

// generateNamespaceManifest generates a namespace manifest for the given namespace.
func generateNamespaceManifest(namespace string) (workv1.Manifest, error) {
	yamlTemplate := `apiVersion: v1
	kind: Namespace
	metadata:
 	name: {{ .NamespaceName }}`

	bodyStringTrimmed := strings.ReplaceAll(yamlTemplate, "\t", "")
	tmpl, err := template.New("namespace").Parse(bodyStringTrimmed)
	if err != nil {
		return workv1.Manifest{}, fmt.Errorf("error parsing template: %v", err)
	}

	namespacePayload := struct {
		NamespaceName string
	}{
		NamespaceName: namespace,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, namespacePayload); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error executing template: %v", err)
	}

	var manifest workv1.Manifest
	if err := yaml.Unmarshal(buf.Bytes(), &manifest); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error unmarshaling manifest: %v", err)
	}
	return manifest, nil
}

// UpdateDeploymentAttributes updates the attributes of the manifests for a deployment based on the remediation type.
func updateDeploymentAttributes(j *Job) (*Job, error) {

	subType := j.SubType
	manifestWork, err := fetchManifestWork(j.Target.ClusterName, j.Resource.ResourceName, nil)
	if err != nil {
		logErrorAndSetJobState("Error obtaining applied ManifestWork status", j, Degraded)
		return nil, err
	}

	manifests := manifestWork.Spec.Workload.Manifests

	updatedManifests := make([]workv1.Manifest, 0, len(manifests))

	//Since we are creating a new slice of manifests, we need to set up the namespace as well
	namespaceManifest, err := generateNamespaceManifest(j.Namespace)
	if err != nil {
		logErrorAndSetJobState("Error generating namespace manifest", j, Degraded)
		return nil, err
	}
	updatedManifests = append(updatedManifests, namespaceManifest)

	for _, manifest := range manifests {

		yamlBytes, err := yamlEncode.Marshal(manifest)
		if err != nil {
			return nil, fmt.Errorf("error encoding manifest: %v", err)
		}
		obj, err := decodeYAMLToObject(string(yamlBytes))
		if err != nil {
			return nil, fmt.Errorf("error decoding manifest: %v", err)
		}

		var updatedManifest *workv1.Manifest
		switch subType {
		case ScaleUp, ScaleDown:
			updatedManifest, err = updateReplicaCount(obj, subType)
		case ScaleOut, ScaleIn:
			updatedManifest, err = updateResourceRequirements(obj, subType)
		default:
			err = fmt.Errorf("unsupported subType: %v", subType)
		}

		if err != nil {
			return nil, fmt.Errorf("error updating manifest: %v", err)
		}

		if updatedManifest != nil {
			updatedManifests = append(updatedManifests, *updatedManifest)
		}
	}

	manifestWork.Spec.Workload.Manifests = updatedManifests
	logs.Logger.Print("-----BEFORE UPDATE---------------")
	logs.Logger.Printf("Updated CPU: %v, Memory: %v\n", manifestWork.Spec.Workload.Manifests[1].RawExtension.Object.(*appsv1.Deployment).Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU], manifestWork.Spec.Workload.Manifests[1].RawExtension.Object.(*appsv1.Deployment).Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory])
	logs.Logger.Print("--------------------")

	updatedManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Update(context.TODO(), manifestWork, metav1.UpdateOptions{})

	if err != nil {
		logErrorAndSetJobState("Error updating ManifestWork", j, Degraded)
		return nil, err
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// appliedManifestWork, err := waitForAppliedManifestWork(updatedManifestWork.Namespace, updatedManifestWork.Name, ctx)
	// if err != nil {
	// 	logs.Logger.Println("Error obtaining applied ManifestWork status:", err)
	// 	j.State = Degraded
	// 	return nil, err
	// }

	j.UpdateJobResource(updatedManifestWork)

	return j, nil
}

// updateNamespaceAndAnnotations updates the namespace and annotations of a Manifest Work object.
func updateNamespaceAndAnnotations(obj runtime.Object, namespace, appName, componentName, instanceID, manifestID string) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		logs.Logger.Fatalf("Failed to get metadata accessor: %v", err)
	}
	metaObj.SetNamespace(namespace)

	annotations := metaObj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["app.icos.eu/name"] = appName
	annotations["app.icos.eu/component"] = componentName
	annotations["app.icos.eu/instance"] = instanceID
	annotations["jobmanager.icos.eu/manifest"] = manifestID
	metaObj.SetAnnotations(annotations)
}

// Deployment Attribute Updates
// ------------------------------------------------

// updateReplicaCount updates the replica count of the deployment based on the remediation type.
func updateReplicaCount(obj runtime.Object, subType RemediationType) (*workv1.Manifest, error) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil, nil
	}

	replicaNumber := *deployment.Spec.Replicas
	horizontalPodAutoscaling(subType, &replicaNumber)
	deployment.Spec.Replicas = &replicaNumber

	rawExtension := runtime.RawExtension{Object: obj}
	return &workv1.Manifest{RawExtension: rawExtension}, nil
}

// updateResourceRequirements updates the resource requirements of the deployment based on the remediation type.
func updateResourceRequirements(obj runtime.Object, subType RemediationType) (*workv1.Manifest, error) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil, nil
	}

	resources := deployment.Spec.Template.Spec.Containers[0].Resources
	logs.Logger.Print("------Inside updateResourceRequirements----------")
	logs.Logger.Printf("Current CPU: %v, Memory: %v\n", resources.Requests[corev1.ResourceCPU], resources.Requests[corev1.ResourceMemory])
	verticalPodAutoscaling(subType, &resources)
	deployment.Spec.Template.Spec.Containers[0].Resources = resources

	rawExtension := runtime.RawExtension{Object: obj}
	return &workv1.Manifest{RawExtension: rawExtension}, nil
}

// horizontalPodAutoscaling adjusts the replica count for scale up or scale down operations.
func horizontalPodAutoscaling(subType RemediationType, replicas *int32) {
	switch subType {
	case ScaleUp:
		*replicas++
	case ScaleDown:
		*replicas--
	}
}

// verticalPodAutoscaling adjusts the resource requirements for scale out or scale in operations.
func verticalPodAutoscaling(subType RemediationType, resources *corev1.ResourceRequirements) {
	// who is responsible for specifying the resource amount?
	cpuAdjustment := int64(1000)                  // in millicores
	memoryAdjustment := int64(1000 * 1024 * 1024) // in bytes (1000 MiB)
	switch subType {
	case ScaleOut:
		adjustResource(corev1.ResourceCPU, resources, cpuAdjustment)
		adjustResource(corev1.ResourceMemory, resources, memoryAdjustment)
	case ScaleIn:
		adjustResource(corev1.ResourceCPU, resources, -cpuAdjustment)
		adjustResource(corev1.ResourceMemory, resources, -memoryAdjustment)
	}
}

// adjustResource adjusts the resource quantity based on the resource name and adjustment value.
func adjustResource(resourceName corev1.ResourceName, resources *corev1.ResourceRequirements, adjustment int64) {
	currentQuantity := resources.Requests[resourceName]
	newQuantity := currentQuantity.DeepCopy()

	if resourceName == corev1.ResourceCPU {
		newQuantity.Add(*resource.NewMilliQuantity(adjustment, resource.DecimalSI))
	} else if resourceName == corev1.ResourceMemory {
		newQuantity.Add(*resource.NewQuantity(adjustment, resource.BinarySI))
	}

	if newQuantity.Sign() >= 0 {
		resources.Requests[resourceName] = newQuantity
	}
}

// Utility Functions
// ------------------------------------------------

// decodeYAMLToObject decodes a YAML string into a runtime object.
func decodeYAMLToObject(yamlString string) (runtime.Object, error) {
	scheme := runtime.NewScheme()
	appsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	codecFactory := serializer.NewCodecFactory(scheme)
	decoder := codecFactory.UniversalDeserializer()

	obj, _, err := decoder.Decode([]byte(yamlString), nil, nil)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// logErrorAndSetJobState logs an error message and sets the job state to the specified state.
func logErrorAndSetJobState(message string, j *Job, state JobState) {
	logs.Logger.Println(message)
	j.State = state
}

// getJobTypeString returns the string representation of a job type.
func getJobTypeString(jobType JobType) string {
	jobString, exists := JobTypeToString[jobType]
	if !exists {
		return "unknownJobType"
	}
	return jobString
}

// FetchNodeUID fetches the UID of a ClusterManager CRD in the hub cluster based on the node name.
func FetchClusterManagerUID(clusterManagerOperatorName string) (string, error) {
	cm, err := clientOperator.ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	uuid, err := uuid.Parse(string(cm.UID))
	if err != nil {
		return "", err
	}

	return uuid.String(), nil
}

// Job Struct Methods
// ------------------------------------------------

func (j *Job) PromoteJob(authHeader string, OwnerID string) error {
	logs.Logger.Println("promoting job...")

	// Create the JSON payload
	payload := struct {
		OwnerID string `json:"owner_id"`
	}{
		OwnerID: OwnerID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logs.Logger.Println("Error marshaling JSON payload:", err)
		return err
	}

	// Create the request with JSON payload
	reqState, err := http.NewRequest("PATCH", jobmanagerBaseURL+"jobmanager/jobs/promote/"+j.ID, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logs.Logger.Println("Error creating Job locking request:", err)
		return err
	}
	reqState.Header.Add("Authorization", authHeader)
	reqState.Header.Add("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(reqState)
	if err != nil {
		logs.Logger.Println("Error occurred during Job locking request:", err)
		return err
	}
	defer resp.Body.Close()

	logs.Logger.Println("GET Lock Response", resp.Status)
	return nil
}

func (j *Job) StateMapper(state workv1.ManifestWorkStatus) {
	offset := len(state.Conditions)
	if offset >= 1 {
		offset = len(state.Conditions) - 1
	}
	switch jobState := state.Conditions[offset].Type; jobState {
	case "Progressing":
		j.State = Progressing
	case "Available":
		j.State = Available
	case "Degraded":
		j.State = Degraded
	default:
		j.State = Applied
	}
}

func (j *Job) UpdateJobResource(manifestWork *workv1.ManifestWork) {
	if manifestWork != nil {
		if len(manifestWork.Status.Conditions) != 0 {
			j.StateMapper(manifestWork.Status)
		} else {
			j.State = Progressing
		}
		j.Resource.ResourceUUID = string(manifestWork.UID)
		j.Resource.ResourceName = manifestWork.Name
		j.Resource.Conditions = append(j.Resource.Conditions, manifestWork.Status.Conditions...)
	} else {
		j.State = Applied
		//concatenate DELETED literal?
		j.Resource.ResourceName = ""
		j.Resource.Conditions = append(j.Resource.Conditions, metav1.Condition{Type: "Applied"})
		// update job status to applied if the manifest work is deleted
		// j.Resource.Conditions = append(j.Resource.Conditions, workv1.ManifestWorkCondition{Type: "Applied"})
	}
	logs.Logger.Printf("Job's Resource details: %#v", j.Resource)
}
