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

package v2beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AtomSpec defines the desired state of Atom.
type AtomSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	General    General     `json:"general"`
	Service    AtomService `json:"service"`
	Kubernetes *Kubernetes `json:"kubernetes,omitempty"`
}

// AtomService is the struct for all service level fields
type AtomService struct {
	Title              string    `json:"title"`
	Subtitle           string    `json:"subtitle"`
	MetadataIdentifier string    `json:"metadataIdentifier"`
	Rights             string    `json:"rights"`
	Updated            *string   `json:"updated,omitempty"` // deprecated
	Author             Author    `json:"author"`
	Datasets           []Dataset `json:"datasets"`
}

// AtomStatus defines the observed state of Atom.
type AtomStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:skipversion
// versionName=v2beta1

// Atom is the Schema for the atoms API.
type Atom struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AtomSpec   `json:"spec,omitempty"`
	Status AtomStatus `json:"status,omitempty"`
}

// Author is the struct with the input for the author field of an atom
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Dataset is the struct for all dataset level fields
type Dataset struct {
	Name               string      `json:"name"`
	Title              string      `json:"title"`
	Subtitle           string      `json:"subtitle"`
	MetadataIdentifier string      `json:"metadataIdentifier"`
	SourceIdentifier   string      `json:"sourceIdentifier"`
	Links              []OtherLink `json:"links,omitempty"`
	Downloads          []Download  `json:"downloads"`
	Bbox               Bbox        `json:"bbox"`
}

// Bbox is the struct for the bounding box extent of an atom
type Bbox struct {
	Minx float32 `json:"minx"`
	Maxx float32 `json:"maxx"`
	Miny float32 `json:"miny"`
	Maxy float32 `json:"maxy"`
}

// Download is the struct for the download level fields
type Download struct {
	Name    string  `json:"name"`
	Title   *string `json:"title,omitempty"`
	Updated *string `json:"updated,omitempty"`
	Content *string `json:"content,omitempty"`
	Links   []Link  `json:"links,omitempty"`
	Srs     Srs     `json:"srs"`
}

// Link represents a link in a download entry
type Link struct {
	BlobKey *string `json:"blobKey"`
	Updated *string `json:"updated,omitempty"`
	Version *string `json:"version,omitempty"`
	Bbox    *Bbox   `json:"bbox,omitempty"`
	Rel     *string `json:"rel,omitempty"`
}

// OtherLink represents any type of link that is not a download link related to the data (see Link)
type OtherLink struct {
	Type        string  `json:"type"`
	URI         string  `json:"uri"`
	ContentType *string `json:"contentType,omitempty"`
	Language    *string `json:"language,omitempty"`
}

// Srs is the struct with the information for the srs field of an atom
type Srs struct {
	URI  string `json:"uri"`
	Code string `json:"code"`
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
