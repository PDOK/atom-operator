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
	"log"
	"net/url"
	"strconv"
	"time"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this Atom (v2beta1) to the Hub version (v3).
func (a *Atom) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.Atom)
	log.Printf("ConvertTo: Converting Atom from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s", a.Namespace, a.Name)

	return a.ToV3(dst)
}

// ConvertTo converts this Atom (v2beta1) to the Hub version (v3).
//
//nolint:cyclop,funlen
func (a *Atom) ToV3(dst *pdoknlv3.Atom) error {
	// ObjectMeta
	dst.ObjectMeta = a.ObjectMeta

	// Lifecycle
	if a.Spec.Kubernetes != nil && a.Spec.Kubernetes.Lifecycle != nil && a.Spec.Kubernetes.Lifecycle.TTLInDays != nil {
		dst.Spec.Lifecycle = &smoothoperatormodel.Lifecycle{
			TTLInDays: GetInt32Pointer(int32(*a.Spec.Kubernetes.Lifecycle.TTLInDays)), //nolint:gosec
		}
	}

	baseURL, err := createBaseURL(pdoknlv3.GetBaseURL(), a.Spec.General)
	if err != nil {
		return err
	}

	// Service
	dst.Spec.Service = pdoknlv3.Service{
		BaseURL:      *baseURL,
		Lang:         "nl",
		Title:        a.Spec.Service.Title,
		Subtitle:     a.Spec.Service.Subtitle,
		OwnerInfoRef: "pdok",
		ServiceMetadataLinks: &pdoknlv3.MetadataLink{
			MetadataIdentifier: a.Spec.Service.MetadataIdentifier,
			Templates:          []string{"csw", "opensearch", "html"},
		},
		Rights: a.Spec.Service.Rights,
	}

	dst.Spec.Service.DatasetFeeds = []pdoknlv3.DatasetFeed{}
	for _, srcDataset := range a.Spec.Service.Datasets {
		dstDatasetFeed := pdoknlv3.DatasetFeed{
			TechnicalName: srcDataset.Name,
			Title:         srcDataset.Title,
			Subtitle:      srcDataset.Subtitle,
			DatasetMetadataLinks: &pdoknlv3.MetadataLink{
				MetadataIdentifier: srcDataset.MetadataIdentifier,
				Templates:          []string{"csw", "html"},
			},
			Author:                            smoothoperatormodel.Author{Name: a.Spec.Service.Author.Name, Email: a.Spec.Service.Author.Email},
			SpatialDatasetIdentifierCode:      smoothutil.Pointer(srcDataset.SourceIdentifier),
			SpatialDatasetIdentifierNamespace: smoothutil.Pointer("http://www.pdok.nl"),
		}

		// Map the links
		for _, srcLink := range srcDataset.Links {
			href, err := smoothoperatormodel.ParseURL(srcLink.URI)
			if err != nil {
				return err
			}
			dstLink := pdoknlv3.Link{
				Rel:   "describedby",
				Title: &srcLink.Type,
				Href:  smoothoperatormodel.URL{URL: href},
			}
			if srcLink.ContentType != nil {
				dstLink.Type = *srcLink.ContentType
			}
			if srcLink.Language != nil {
				dstLink.Hreflang = srcLink.Language
			}

			dstDatasetFeed.Links = append(dstDatasetFeed.Links, dstLink)
		}

		// Map the entries
		for _, srcDownload := range srcDataset.Downloads {

			uri, err := smoothoperatormodel.ParseURL(srcDownload.Srs.URI)
			if err != nil {
				return err
			}

			dstEntry := pdoknlv3.Entry{
				TechnicalName: srcDownload.Name,
				Content:       srcDownload.Content,
				SRS: pdoknlv3.SRS{
					URI:  smoothoperatormodel.URL{URL: uri},
					Name: srcDownload.Srs.Code,
				},
				Polygon: pdoknlv3.Polygon{
					BBox: smoothoperatormodel.BBox{
						MinX: GetFloat32AsString(srcDataset.Bbox.Minx),
						MinY: GetFloat32AsString(srcDataset.Bbox.Miny),
						MaxX: GetFloat32AsString(srcDataset.Bbox.Maxx),
						MaxY: GetFloat32AsString(srcDataset.Bbox.Maxy),
					},
				},
			}

			if srcDownload.Title != nil {
				dstEntry.Title = srcDownload.Title
			}

			var updated string
			if srcDownload.Updated != nil {
				updated = *srcDownload.Updated
			} else if a.Spec.Service.Updated != nil {
				updated = *a.Spec.Service.Updated
			}

			parsedUpdatedTime, err := time.Parse(time.RFC3339, updated)
			if err != nil {
				log.Printf("Error parsing updated time: %v", err)
				dstEntry.Updated = metav1.Now()
			} else {
				updatedTime := metav1.NewTime(parsedUpdatedTime)
				dstEntry.Updated = updatedTime
			}

			// Map the links
			for _, srcLink := range srcDownload.Links {
				dstDownloadLink := pdoknlv3.DownloadLink{}

				if srcLink.BlobKey != nil {
					dstDownloadLink.Data = *srcLink.BlobKey
				}
				if srcLink.Updated != nil {
					dstDownloadLink.Time = srcLink.Updated
				}
				if srcLink.Bbox != nil {
					dstDownloadLink.BBox = &smoothoperatormodel.BBox{
						MinX: GetFloat32AsString(srcLink.Bbox.Minx),
						MinY: GetFloat32AsString(srcLink.Bbox.Miny),
						MaxX: GetFloat32AsString(srcLink.Bbox.Maxx),
						MaxY: GetFloat32AsString(srcLink.Bbox.Maxy),
					}
				}
				if srcLink.Rel != nil {
					dstDownloadLink.Rel = srcLink.Rel
				}

				dstEntry.DownloadLinks = append(dstEntry.DownloadLinks, dstDownloadLink)
			}

			dstDatasetFeed.Entries = append(dstDatasetFeed.Entries, dstEntry)
		}

		dst.Spec.Service.DatasetFeeds = append(dst.Spec.Service.DatasetFeeds, dstDatasetFeed)
	}

	return nil
}

// ConvertFrom converts the Hub version (v3) to this Atom (v2beta1).
//
//nolint:funlen
func (a *Atom) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*pdoknlv3.Atom)
	log.Printf("ConvertFrom: Converting Atom from Hub version v3 to Spoke version v2beta1;"+
		"source: %s/%s", src.Namespace, src.Name)

	// ObjectMeta
	a.ObjectMeta = src.ObjectMeta

	// General
	a.Spec.General = General{
		Dataset:      src.Labels["dataset"],
		DatasetOwner: src.Labels["dataset-owner"],
		DataVersion:  nil,
	}

	serviceVersion, ok := src.Labels["service-version"]
	if ok {
		a.Spec.General.ServiceVersion = &serviceVersion
	}

	theme, ok := src.Labels["theme"]
	if ok {
		a.Spec.General.Theme = &theme
	}

	// Service
	a.Spec.Service = AtomService{
		Title:    src.Spec.Service.Title,
		Subtitle: src.Spec.Service.Subtitle,
		Rights:   src.Spec.Service.Rights,
		Author: Author{
			Name:  "PDOK Beheer",
			Email: "beheerPDOK@kadaster.nl",
		},
		MetadataIdentifier: src.Spec.Service.ServiceMetadataLinks.MetadataIdentifier,
	}

	// Datasets
	a.Spec.Service.Datasets = []Dataset{}
	for _, srcDatasetFeed := range src.Spec.Service.DatasetFeeds {
		dstDataset := Dataset{
			Name:               srcDatasetFeed.TechnicalName,
			Title:              srcDatasetFeed.Title,
			Subtitle:           srcDatasetFeed.Subtitle,
			SourceIdentifier:   smoothutil.PointerVal(srcDatasetFeed.SpatialDatasetIdentifierCode, ""),
			MetadataIdentifier: srcDatasetFeed.DatasetMetadataLinks.MetadataIdentifier,
		}

		// Map the links
		for _, srcLink := range srcDatasetFeed.Links {
			dstDataset.Links = append(dstDataset.Links, OtherLink{
				Type:        smoothutil.PointerVal(srcLink.Title, ""),
				URI:         srcLink.Href.String(),
				ContentType: &srcLink.Type,
				Language:    srcLink.Hreflang,
			})
		}

		if len(srcDatasetFeed.Entries) > 0 {
			// We can assume all entries have the same bbox, so we take the first one
			firstBbox := srcDatasetFeed.Entries[0].Polygon.BBox
			dstDataset.Bbox = Bbox{
				Minx: GetStringAsFloat32(firstBbox.MinX),
				Miny: GetStringAsFloat32(firstBbox.MinY),
				Maxx: GetStringAsFloat32(firstBbox.MaxX),
				Maxy: GetStringAsFloat32(firstBbox.MaxY),
			}
		}

		// Map the downloads
		for _, srcEntry := range srcDatasetFeed.Entries {
			dstDownload := Download{
				Name:    srcEntry.TechnicalName,
				Title:   srcEntry.Title,
				Content: srcEntry.Content,
			}

			updatedString := srcEntry.Updated.Format(time.RFC3339)
			dstDownload.Updated = &updatedString

			dstDownload.Srs = Srs{
				URI:  srcEntry.SRS.URI.String(),
				Code: srcEntry.SRS.Name,
			}

			// Map the links
			for _, srcDownloadLink := range srcEntry.DownloadLinks {
				dstLink := Link{
					BlobKey: &srcDownloadLink.Data,
				}

				if srcDownloadLink.Rel != nil && *srcDownloadLink.Rel != "" {
					dstLink.Rel = srcDownloadLink.Rel
				}

				if srcDownloadLink.Time != nil {
					dstLink.Updated = srcDownloadLink.Time
				}
				if srcDownloadLink.BBox != nil {
					dstLink.Bbox = &Bbox{
						Minx: GetStringAsFloat32(srcDownloadLink.BBox.MinX),
						Miny: GetStringAsFloat32(srcDownloadLink.BBox.MinY),
						Maxx: GetStringAsFloat32(srcDownloadLink.BBox.MaxX),
						Maxy: GetStringAsFloat32(srcDownloadLink.BBox.MaxY),
					}
				}
				dstDownload.Links = append(dstDownload.Links, dstLink)
			}

			dstDataset.Downloads = append(dstDataset.Downloads, dstDownload)
		}
		a.Spec.Service.Datasets = append(a.Spec.Service.Datasets, dstDataset)
	}

	// Kubernetes
	a.Spec.Kubernetes = &Kubernetes{
		Lifecycle: &Lifecycle{},
	}
	if src.Spec.Lifecycle != nil && src.Spec.Lifecycle.TTLInDays != nil {
		a.Spec.Kubernetes.Lifecycle.TTLInDays = GetIntPointer(int(*src.Spec.Lifecycle.TTLInDays))
	}

	return nil
}

func createBaseURL(host string, general General) (*smoothoperatormodel.URL, error) {
	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	baseURL = baseURL.JoinPath(general.DatasetOwner, general.Dataset)
	if general.Theme != nil {
		baseURL = baseURL.JoinPath(*general.Theme)
	}
	baseURL = baseURL.JoinPath("atom")

	if general.ServiceVersion != nil {
		baseURL = baseURL.JoinPath(*general.ServiceVersion)
	}

	return &smoothoperatormodel.URL{URL: baseURL}, nil
}

func GetInt32Pointer(value int32) *int32 {
	return &value
}

func GetIntPointer(value int) *int {
	return &value
}

func GetFloat32AsString(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}

func GetStringAsFloat32(value string) float32 {
	float, _ := strconv.ParseFloat(value, 32)
	return float32(float)
}
