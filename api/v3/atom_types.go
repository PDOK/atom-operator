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
	"strings"

	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var baseURL string
var host string
var blobEndpoint string

// AtomSpec defines the desired state of Atom.
type AtomSpec struct {
	Lifecycle    smoothoperatormodel.Lifecycle `json:"lifecycle,omitempty"`
	Service      Service                       `json:"service"`
	DatasetFeeds []DatasetFeed                 `json:"datasetFeeds,omitempty"`
	//+kubebuilder:validation:Type=object
	//+kubebuilder:validation:Schemaless
	//+kubebuilder:pruning:PreserveUnknownFields
	// Optional strategic merge patch for the pod in the deployment. E.g. to patch the resources or add extra env vars.
	PodSpecPatch *corev1.PodSpec `json:"podSpecPatch,omitempty"`
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
	TechnicalName                     string       `json:"technicalName"`
	Title                             string       `json:"title"`
	Subtitle                          string       `json:"subtitle,omitempty"`
	Links                             []Link       `json:"links,omitempty"`
	DatasetMetadataLinks              MetadataLink `json:"datasetMetadataLinks,omitempty"`
	Author                            Author       `json:"author,omitempty"`
	SpatialDatasetIdentifierCode      string       `json:"spatial_dataset_identifier_code,omitempty"`      //nolint:tagliatelle // This is according to Atom spec
	SpatialDatasetIdentifierNamespace string       `json:"spatial_dataset_identifier_namespace,omitempty"` //nolint:tagliatelle // This is according to Atom spec
	Entries                           []Entry      `json:"entries,omitempty"`
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
	Data    string                    `json:"data"`
	Rel     string                    `json:"rel,omitempty"`
	Version *string                   `json:"version,omitempty"`
	Time    *string                   `json:"time,omitempty"`
	BBox    *smoothoperatormodel.BBox `json:"bbox,omitempty"`
}

// Polygon describes the bounding box of an entry or download
type Polygon struct {
	BBox smoothoperatormodel.BBox `json:"bbox"`
}

// SRS describes the Spatial Reference System for an entry
type SRS struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status
// versionName=v3
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=pdok

// Atom is the Schema for the atoms API.
type Atom struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AtomSpec                           `json:"spec,omitempty"`
	Status smoothoperatormodel.OperatorStatus `json:"status,omitempty"`
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

func SetBaseURL(atomBaseURL string) {
	baseURL = strings.TrimSuffix(atomBaseURL, "/")
}

func GetBaseURL() string {
	return baseURL
}

func SetHost(atomHost string) {
	host = strings.TrimSuffix(atomHost, "/")
}

func GetHost() string {
	return host
}

func SetBlobEndpoint(atomBlobEndpoint string) {
	blobEndpoint = atomBlobEndpoint
}

func GetBlobEndpoint() string {
	return blobEndpoint
}

func (r *Atom) GetURI() (uri string) {
	datasetOwner := "unknown"
	if v, ok := r.ObjectMeta.Labels["dataset-owner"]; ok {
		datasetOwner = v
	}
	dataset := "unknown"
	if v, ok := r.ObjectMeta.Labels["dataset"]; ok {
		dataset = v
	}
	uri = fmt.Sprintf("%s/%s", datasetOwner, dataset)

	if v, ok := r.ObjectMeta.Labels["theme"]; ok {
		uri += "/" + v
	}
	uri += "/atom"
	if v, ok := r.ObjectMeta.Labels["service-version"]; ok {
		uri += "/" + v
	}
	return
}

func (r *Atom) GetIndexedDownloadLinks() (downloadLinks map[int8]DownloadLink) {
	downloadLinks = make(map[int8]DownloadLink)
	var index int8
	for _, datasetFeed := range r.Spec.DatasetFeeds {
		for _, entry := range datasetFeed.Entries {
			for _, downloadLink := range entry.DownloadLinks {
				downloadLinks[index] = downloadLink
				index++
			}
		}
	}
	return
}

func (dl *DownloadLink) GetBlobPrefix() string {
	index := strings.LastIndex(dl.Data, "/")
	return dl.Data[:index]
}

func (dl *DownloadLink) GetBlobName() string {
	index := strings.LastIndex(dl.Data, "/") + 1
	return dl.Data[index:]
}
