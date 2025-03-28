//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v3

import (
	"github.com/pdok/smooth-operator/model"
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Atom) DeepCopyInto(out *Atom) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Atom.
func (in *Atom) DeepCopy() *Atom {
	if in == nil {
		return nil
	}
	out := new(Atom)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Atom) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AtomList) DeepCopyInto(out *AtomList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Atom, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AtomList.
func (in *AtomList) DeepCopy() *AtomList {
	if in == nil {
		return nil
	}
	out := new(AtomList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AtomList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AtomSpec) DeepCopyInto(out *AtomSpec) {
	*out = *in
	in.Lifecycle.DeepCopyInto(&out.Lifecycle)
	in.Service.DeepCopyInto(&out.Service)
	if in.PodSpecPatch != nil {
		in, out := &in.PodSpecPatch, &out.PodSpecPatch
		*out = new(v1.PodSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AtomSpec.
func (in *AtomSpec) DeepCopy() *AtomSpec {
	if in == nil {
		return nil
	}
	out := new(AtomSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatasetFeed) DeepCopyInto(out *DatasetFeed) {
	*out = *in
	if in.Links != nil {
		in, out := &in.Links, &out.Links
		*out = make([]Link, len(*in))
		copy(*out, *in)
	}
	in.DatasetMetadataLinks.DeepCopyInto(&out.DatasetMetadataLinks)
	out.Author = in.Author
	if in.Entries != nil {
		in, out := &in.Entries, &out.Entries
		*out = make([]Entry, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatasetFeed.
func (in *DatasetFeed) DeepCopy() *DatasetFeed {
	if in == nil {
		return nil
	}
	out := new(DatasetFeed)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DownloadLink) DeepCopyInto(out *DownloadLink) {
	*out = *in
	if in.Version != nil {
		in, out := &in.Version, &out.Version
		*out = new(string)
		**out = **in
	}
	if in.Time != nil {
		in, out := &in.Time, &out.Time
		*out = new(string)
		**out = **in
	}
	if in.BBox != nil {
		in, out := &in.BBox, &out.BBox
		*out = new(model.BBox)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DownloadLink.
func (in *DownloadLink) DeepCopy() *DownloadLink {
	if in == nil {
		return nil
	}
	out := new(DownloadLink)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Entry) DeepCopyInto(out *Entry) {
	*out = *in
	if in.DownloadLinks != nil {
		in, out := &in.DownloadLinks, &out.DownloadLinks
		*out = make([]DownloadLink, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Updated != nil {
		in, out := &in.Updated, &out.Updated
		*out = (*in).DeepCopy()
	}
	if in.Polygon != nil {
		in, out := &in.Polygon, &out.Polygon
		*out = new(Polygon)
		**out = **in
	}
	if in.SRS != nil {
		in, out := &in.SRS, &out.SRS
		*out = new(SRS)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Entry.
func (in *Entry) DeepCopy() *Entry {
	if in == nil {
		return nil
	}
	out := new(Entry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Link) DeepCopyInto(out *Link) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Link.
func (in *Link) DeepCopy() *Link {
	if in == nil {
		return nil
	}
	out := new(Link)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetadataLink) DeepCopyInto(out *MetadataLink) {
	*out = *in
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetadataLink.
func (in *MetadataLink) DeepCopy() *MetadataLink {
	if in == nil {
		return nil
	}
	out := new(MetadataLink)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Polygon) DeepCopyInto(out *Polygon) {
	*out = *in
	out.BBox = in.BBox
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Polygon.
func (in *Polygon) DeepCopy() *Polygon {
	if in == nil {
		return nil
	}
	out := new(Polygon)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SRS) DeepCopyInto(out *SRS) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SRS.
func (in *SRS) DeepCopy() *SRS {
	if in == nil {
		return nil
	}
	out := new(SRS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Service) DeepCopyInto(out *Service) {
	*out = *in
	in.ServiceMetadataLinks.DeepCopyInto(&out.ServiceMetadataLinks)
	if in.DatasetFeeds != nil {
		in, out := &in.DatasetFeeds, &out.DatasetFeeds
		*out = make([]DatasetFeed, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.Author = in.Author
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Service.
func (in *Service) DeepCopy() *Service {
	if in == nil {
		return nil
	}
	out := new(Service)
	in.DeepCopyInto(out)
	return out
}
