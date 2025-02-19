package v2beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Status - The status for custom resources managed by the operator-sdk.
type Status struct {
	Conditions []Condition `json:"conditions,omitempty"`
	Deployment *string     `json:"deployment,omitempty"`
	Resources  []Resources `json:"resources,omitempty"`
}

// Resources is the struct for the resources field within status
type Resources struct {
	APIVersion *string `json:"apiversion,omitempty"`
	Kind       *string `json:"kind,omitempty"`
	Name       *string `json:"name,omitempty"`
}

// General is the struct with all generic fields for the crds
type General struct {
	Dataset        string  `json:"dataset"`
	Theme          *string `json:"theme,omitempty"`
	DatasetOwner   string  `json:"datasetOwner"`
	ServiceVersion *string `json:"serviceVersion,omitempty"`
	DataVersion    *string `json:"dataVersion,omitempty"`
}

// Kubernetes is the struct with all fields that can be defined in kubernetes fields in the crds
type Kubernetes struct {
	Autoscaling *Autoscaling                 `json:"autoscaling,omitempty"`
	HealthCheck *HealthCheck                 `json:"healthCheck,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
	Lifecycle   *Lifecycle                   `json:"lifecycle,omitempty"`
}

// Autoscaling is the struct with all fields to configure autoscalers for the crs
type Autoscaling struct {
	AverageCPUUtilization *int `json:"averageCpuUtilization,omitempty"`
	MinReplicas           *int `json:"minReplicas,omitempty"`
	MaxReplicas           *int `json:"maxReplicas,omitempty"`
}

// HealthCheck is the struct with all fields to configure healthchecks for the crs
type HealthCheck struct {
	Querystring *string `json:"querystring,omitempty"`
	Mimetype    *string `json:"mimetype,omitempty"`
	Boundingbox *string `json:"boundingbox,omitempty"`
}

// Lifecycle is the struct with the fields to configure lifecycle settings for the resources
type Lifecycle struct {
	TTLInDays *int `json:"ttlInDays,omitempty"`
}

// TODO Should we move this to an ansible package?

// Condition - the condition for the ansible operator
type Condition struct {
	Type               ConditionType   `json:"type"`
	Status             ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time     `json:"lastTransitionTime"`
	AnsibleResult      *ResultAnsible  `json:"ansibleResult,omitempty"`
	Reason             string          `json:"reason"`
	Message            string          `json:"message"`
}

// ConditionType specifies a string for field ConditionType
type ConditionType string

// ConditionStatus specifies a string for field ConditionType
type ConditionStatus string

// // This const specifies allowed fields for Status
// const (
// 	ConditionTrue    ConditionStatus = "True"
// 	ConditionFalse   ConditionStatus = "False"
// 	ConditionUnknown ConditionStatus = "Unknown"
// )

// ResultAnsible - encapsulation of the ansible result. 'AnsibleResult' is turned around in struct to comply with linting
type ResultAnsible struct {
	Ok               int    `json:"ok"`
	Changed          int    `json:"changed"`
	Skipped          int    `json:"skipped"`
	Failures         int    `json:"failures"`
	TimeOfCompletion string `json:"completion"`
}
