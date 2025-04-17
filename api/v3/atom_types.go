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
	"net/url"
	"strings"

	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var baseURL string
var host string
var blobEndpoint string

// AtomSpec defines the desired state of Atom.
type AtomSpec struct {
	Lifecycle *smoothoperatormodel.Lifecycle `json:"lifecycle,omitempty"`
	Service   Service                        `json:"service"`
}

// Service defines the service configuration for the Atom feed
type Service struct {
	BaseURL              string        `json:"baseUrl"`
	Lang                 string        `json:"lang"` // TODO default nl
	Stylesheet           *string       `json:"stylesheet,omitempty"`
	Title                string        `json:"title"`
	Subtitle             string        `json:"subtitle"`
	OwnerInfoRef         string        `json:"ownerInfoRef"`
	ServiceMetadataLinks *MetadataLink `json:"serviceMetadataLinks,omitempty"`
	Links                []Link        `json:"links,omitempty"` // TODO minlength 1 if not nil (zal momenteel altijd nil zijn)
	Rights               string        `json:"rights"`
	DatasetFeeds         []DatasetFeed `json:"datasetFeeds"` // TODO minlength 1
}

// Link represents a link in the service or dataset feed
type Link struct {
	Href     string  `json:"href"`
	Rel      string  `json:"rel"`
	Type     string  `json:"type"`
	Hreflang *string `json:"hreflang,omitempty"`
	Title    *string `json:"title,omitempty"`
}

// DatasetFeed represents individual dataset feeds within the Atom service
type DatasetFeed struct {
	TechnicalName                     string                     `json:"technicalName"`
	Title                             string                     `json:"title"`
	Subtitle                          string                     `json:"subtitle"`
	Links                             []Link                     `json:"links,omitempty"` // TODO minlength 1 if not nil
	DatasetMetadataLinks              *MetadataLink              `json:"datasetMetadataLinks,omitempty"`
	Author                            smoothoperatormodel.Author `json:"author"`
	SpatialDatasetIdentifierCode      *string                    `json:"spatialDatasetIdentifierCode,omitempty"`
	SpatialDatasetIdentifierNamespace *string                    `json:"spatialDatasetIdentifierNamespace,omitempty"`
	Entries                           []Entry                    `json:"entries"` // TODO minlength 1
}

// MetadataLink represents a link in the service or dataset feed
type MetadataLink struct {
	MetadataIdentifier string   `json:"metadataIdentifier"`
	Templates          []string `json:"templates"` // TODO min 1
}

// Entry represents an entry within a dataset feed, typically for downloads
type Entry struct {
	TechnicalName string         `json:"technicalName"`
	Title         *string        `json:"title,omitempty"`
	Content       *string        `json:"content,omitempty"` // required if downloadLinks >=2
	DownloadLinks []DownloadLink `json:"downloadlinks"`     // TODO minlength 1
	Updated       metav1.Time    `json:"updated"`
	Polygon       Polygon        `json:"polygon"`
	SRS           SRS            `json:"srs"`
}

// DownloadLink specifies download information for entries
type DownloadLink struct {
	Data string                    `json:"data"`
	Rel  *string                   `json:"rel,omitempty"`
	Time *string                   `json:"time,omitempty"`
	BBox *smoothoperatormodel.BBox `json:"bbox,omitempty"`
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

	Spec   AtomSpec                            `json:"spec"`
	Status *smoothoperatormodel.OperatorStatus `json:"status,omitempty"`
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

func (r *Atom) GetBaseURLPath() string {
	url, _ := url.Parse(r.Spec.Service.BaseURL)
	return strings.Replace(url.Path, "/", "", 1)
}

func (r *Atom) GetIndexedDownloadLinks() (downloadLinks map[int8]DownloadLink) {
	downloadLinks = make(map[int8]DownloadLink)
	var index int8
	for _, datasetFeed := range r.Spec.Service.DatasetFeeds {
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
