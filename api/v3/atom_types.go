/*
MIT License

Copyright (c) 2024 Publieke Dienstverlening op de Kaart

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v3

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BaseURLHost is accessed by other api versions (i.e. v2beta1)
var baseURLHost string

// AtomSpec defines the desired state of Atom.
type AtomSpec struct {
	Lifecycle    Lifecycle     `json:"lifecycle,omitempty"`
	Service      Service       `json:"service"`
	DatasetFeeds []DatasetFeed `json:"datasetFeeds,omitempty"`
	//+kubebuilder:validation:Type=object
	//+kubebuilder:validation:Schemaless
	//+kubebuilder:pruning:PreserveUnknownFields
	// Optional strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch *corev1.PodSpec `json:"podSpecPatch,omitempty"`
}

// todo: move to higher level (operator-support repo)
type Lifecycle struct {
	TTLInDays *int32 `json:"ttlInDays,omitempty"`
}

// Service defines the service configuration for the Atom feed
type Service struct {
	BaseURL              string       `json:"baseUrl"`
	Lang                 string       `json:"lang,omitempty"`
	Stylesheet           string       `json:"stylesheet,omitempty"`
	Title                string       `json:"title"`
	Subtitle             string       `json:"subtitle,omitempty"`
	OwnerInfoRef         string       `json:"ownerInfoRef"`
	ServiceMetadataLinks MetadataLink `json:"serviceMetadataLinks,omitempty"`
	Rights               string       `json:"rights,omitempty"`
	Author               Author       `json:"author,omitempty"`
  
	// +kubebuilder:validation:Optional
	GeneratorConfig string `json:"-"` // Skip this field in the CRD schema
}

// Link represents a link in the service or dataset feed
type Link struct {
	Href     string `json:"href"`
	Category string `json:"category,omitempty"`
	Rel      string `json:"rel,omitempty"`
	Type     string `json:"type,omitempty"`
	Hreflang string `json:"hreflang,omitempty"`
	Title    string `json:"title,omitempty"`
}

// Author specifies the author or owner information
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// DatasetFeed represents individual dataset feeds within the Atom service
type DatasetFeed struct {
	TechnicalName        string       `json:"technicalName"`
	Title                string       `json:"title"`
	Subtitle             string       `json:"subtitle,omitempty"`
	Links                []Link       `json:"links,omitempty"` // Todo kan weg?
	DatasetMetadataLinks MetadataLink `json:"datasetMetadataLinks,omitempty"`
	SpatialDatasetIdentifierCode      string  `json:"spatial_dataset_identifier_code,omitempty"`
	SpatialDatasetIdentifierNamespace string  `json:"spatial_dataset_identifier_namespace,omitempty"`
	Entries                           []Entry `json:"entries,omitempty"`
}

// MetadataLink represents a link in the service or dataset feed
type MetadataLink struct {
	MetadataIdentifier string   `json:"metadataIdentifier"`
	Templates          []string `json:"templates,omitempty"`
}

// Entry represents an entry within a dataset feed, typically for downloads
type Entry struct {
	TechnicalName string         `json:"technicalName"`
	Title         string         `json:"title,omitempty"`
	Content       string         `json:"content,omitempty"`
	DownloadLinks []DownloadLink `json:"downloadlinks,omitempty"`
	Updated       *metav1.Time   `json:"updated,omitempty"`
	Polygon       *Polygon       `json:"polygon,omitempty"`
	SRS           *SRS           `json:"srs,omitempty"`
}

// DownloadLink specifies download information for entries
type DownloadLink struct {
	Data    string  `json:"data"`
	Rel     string  `json:"rel,omitempty"`
	Version *string `json:"version,omitempty"`
	Time    *string `json:"time,omitempty"`
	BBox    *BBox   `json:"bbox,omitempty"`
}

// Polygon describes the bounding box of an entry or download
type Polygon struct {
	BBox BBox `json:"bbox"`
}

// BBox defines a bounding box with coordinates
type BBox struct {
	// Linksboven X coördinaat
	// +kubebuilder:validation:Pattern="^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$"
	MinX string `json:"minx"`
	// Rechtsonder X coördinaat
	// +kubebuilder:validation:Pattern="^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$"
	MaxX string `json:"maxx"`
	// Linksboven Y coördinaat
	// +kubebuilder:validation:Pattern="^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$"
	MinY string `json:"miny"`
	// Rechtsonder Y coördinaat
	// +kubebuilder:validation:Pattern="^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$"
	MaxY string `json:"maxy"`
}

// SRS describes the Spatial Reference System for an entry
type SRS struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// AtomStatus defines the observed state of Atom.
type AtomStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Each condition contains details for one aspect of the current state of this Atom.
	// Known .status.conditions.type are: "Reconciled"
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// The result of creating or updating of each derived resource for this Atom.
	OperationResults map[string]controllerutil.OperationResult `json:"operationResults,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status
// versionName=v3
// +kubebuilder:storageversion

// Atom is the Schema for the atoms API.
type Atom struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AtomSpec   `json:"spec,omitempty"`
	Status AtomStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AtomList contains a list of Atom.
type AtomList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Atom `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Atom{}, &AtomList{})
}

// SetAtomBaseURLHost is used to set the BaseURL Host in main
func SetAtomBaseURLHost(atomBaseURLHost string) {
	baseURLHost = atomBaseURLHost
}

func GetAtomBaseURLHost() string {
	return baseURLHost
}

func (r *Atom) GetURI() (URI string) {
	datasetOwner := "unknown"
	if v, ok := r.ObjectMeta.Labels["dataset-owner"]; ok {
		datasetOwner = v
	}
	dataset := "unknown"
	if v, ok := r.ObjectMeta.Labels["dataset"]; ok {
		dataset = v
	}
	URI = fmt.Sprintf("%s/%s", datasetOwner, dataset)

	if v, ok := r.ObjectMeta.Labels["theme"]; ok {
		URI += fmt.Sprintf("/%s", v)
	}
	URI += "/atom"
	if v, ok := r.ObjectMeta.Labels["service-version"]; ok {
		URI += fmt.Sprintf("/%s", v)
	}
	return
}

func (r *Atom) GetNrOfDownloadLinks() (count int) {
	for _, datasetFeed := range r.Spec.DatasetFeeds {
		for _, entry := range datasetFeed.Entries {
			for range entry.DownloadLinks {
				count++
			}
		}
	}
	return
}

func (dl *DownloadLink) GetBlobPrefix() string {
	index := strings.LastIndex(dl.Data, "/") + 1
	return dl.Data[:index]
}

func (dl *DownloadLink) GetBlobName() string {
	index := strings.LastIndex(dl.Data, "/") + 1
	return dl.Data[index:]
}
