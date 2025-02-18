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
	"fmt"
	"log"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this Atom (v2beta1) to the Hub version (v3).
func (src *Atom) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.Atom)
	log.Printf("ConvertTo: Converting Atom from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	host := "https://service.dok.nl/" // Todo read from flag

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Lifecycle
	log.Printf("Start mapping the Lifecycle specs...")
	if src.Spec.Kubernetes != nil && src.Spec.Kubernetes.Lifecycle != nil && src.Spec.Kubernetes.Lifecycle.TTLInDays != nil {
		dst.Spec.Lifecycle.TTLInDays = GetInt32Pointer(int32(*src.Spec.Kubernetes.Lifecycle.TTLInDays))
	}
	log.Printf("Done mapping the Lifecycle specs...")

	// Service
	log.Printf("Start mapping the Service...")
	dst.Spec.Service = pdoknlv3.Service{
		BaseURL:      createBaseURL(host, src.Spec.General),
		Lang:         "nl",
		Stylesheet:   "https://service.pdok.nl/atom/style/style.xsl",
		Title:        src.Spec.Service.Title,
		Subtitle:     src.Spec.Service.Subtitle,
		OwnerInfoRef: "pdok",
		Links:        []pdoknlv3.Link{},
		Rights:       src.Spec.Service.Rights,
	}
	log.Printf("Done mapping the Service...")

	dst.Spec.DatasetFeeds = []pdoknlv3.DatasetFeed{}
	log.Printf("Start mapping the Datasets...")
	for _, srcDataset := range src.Spec.Service.Datasets {
		dstDatasetFeed := pdoknlv3.DatasetFeed{
			TechnicalName:                     srcDataset.Name,
			Title:                             srcDataset.Title,
			Subtitle:                          srcDataset.Subtitle,
			SpatialDatasetIdentifierCode:      srcDataset.SourceIdentifier,
			SpatialDatasetIdentifierNamespace: "http://www.pdok.nl",
		}

		// Map the links
		log.Printf("Start mapping the Links...")
		for _, srcLink := range srcDataset.Links {
			dstLink := pdoknlv3.Link{
				Title: srcLink.Type,
				Href:  srcLink.URI,
			}
			if srcLink.ContentType != nil {
				dstLink.Type = *srcLink.ContentType
			}
			if srcLink.Language != nil {
				dstLink.Href = *srcLink.Language
			}

			dstDatasetFeed.Links = append(dstDatasetFeed.Links, dstLink)
		}
		log.Printf("Done mapping the Links...")

		// Map the entries
		log.Printf("Start mapping the Entries...")
		for _, srcDownload := range srcDataset.Downloads {
			dstEntry := pdoknlv3.Entry{
				TechnicalName: srcDownload.Name,
				SRS: &pdoknlv3.SRS{
					URI:  srcDownload.Srs.URI,
					Name: srcDownload.Srs.Code,
				},
				Polygon: &pdoknlv3.Polygon{
					BBox: pdoknlv3.BBox{
						MinX: GetFloat32AsString(srcDataset.Bbox.Minx),
						MinY: GetFloat32AsString(srcDataset.Bbox.Miny),
						MaxX: GetFloat32AsString(srcDataset.Bbox.Maxx),
						MaxY: GetFloat32AsString(srcDataset.Bbox.Maxy),
					},
				},
			}

			if srcDownload.Title != nil {
				dstEntry.Title = *srcDownload.Title
			}
			if srcDownload.Content != nil {
				dstEntry.Content = *srcDownload.Content
			}
			if srcDownload.Updated != nil {
				parsedUpdatedTime, err := time.Parse(time.RFC3339, *srcDownload.Updated)
				if err != nil {
					log.Printf("Error parsing updated time: %v", err)
					dstEntry.Updated = nil
				}
				updatedTime := metav1.NewTime(parsedUpdatedTime)
				dstEntry.Updated = &updatedTime
			}

			// Map the links
			log.Printf("Start mapping the DownloadLinks...")
			for _, srcLink := range srcDownload.Links {
				dstDownloadLink := pdoknlv3.DownloadLink{}

				if srcLink.BlobKey != nil {
					dstDownloadLink.Data = *srcLink.BlobKey
				}
				if srcLink.Updated != nil {
					dstDownloadLink.Time = srcLink.Updated
				}
				if srcLink.Version != nil {
					dstDownloadLink.Version = srcLink.Version
				}
				if srcLink.Bbox != nil {
					dstDownloadLink.BBox = &pdoknlv3.BBox{
						MinX: GetFloat32AsString(srcLink.Bbox.Minx),
						MinY: GetFloat32AsString(srcLink.Bbox.Miny),
						MaxX: GetFloat32AsString(srcLink.Bbox.Maxx),
						MaxY: GetFloat32AsString(srcLink.Bbox.Maxy),
					}
				}
				if srcLink.Rel != nil {
					dstDownloadLink.Rel = *srcLink.Rel
				}

				dstEntry.DownloadLinks = append(dstEntry.DownloadLinks, dstDownloadLink)
			}
			log.Printf("Done mapping the DownloadLinks...")

			dstDatasetFeed.Entries = append(dstDatasetFeed.Entries, dstEntry)
		}
		log.Printf("Done mapping the Entries...")

		dst.Spec.DatasetFeeds = append(dst.Spec.DatasetFeeds, dstDatasetFeed)
	}
	log.Printf("Done mapping the Datasets...")

	return nil
}

// ConvertFrom converts the Hub version (v3) to this Atom (v2beta1).
func (dst *Atom) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*pdoknlv3.Atom)
	log.Printf("ConvertFrom: Converting Atom from Hub version v3 to Spoke version v2beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// General
	log.Printf("Start mapping the General specs...")
	dst.Spec.General = General{
		Dataset:      src.ObjectMeta.Labels["dataset"],
		DatasetOwner: src.ObjectMeta.Labels["dataset-owner"],
		DataVersion:  nil,
	}

	serviceVersion, ok := src.ObjectMeta.Labels["service-version"]
	if ok {
		dst.Spec.General.ServiceVersion = &serviceVersion
	}

	theme, ok := src.ObjectMeta.Labels["theme"]
	if ok {
		dst.Spec.General.Theme = &theme
	}

	log.Printf("Done mapping the General specs...")

	// Service
	log.Printf("Start mapping the Service...")
	dst.Spec.Service = AtomService{
		Title:    src.Spec.Service.Title,
		Subtitle: src.Spec.Service.Subtitle,
		Rights:   src.Spec.Service.Rights,
		Author: Author{
			Name:  "PDOK Beheer",
			Email: "beheerPDOK@kadaster.nl",
		},
	}
	log.Printf("Done mapping the Service...")

	// Datasets
	log.Printf("Start mapping the Datasets...")
	dst.Spec.Service.Datasets = []Dataset{}
	for _, srcDatasetFeed := range src.Spec.DatasetFeeds {
		dstDataset := Dataset{
			Name:             srcDatasetFeed.TechnicalName,
			Title:            srcDatasetFeed.Title,
			Subtitle:         srcDatasetFeed.Subtitle,
			SourceIdentifier: srcDatasetFeed.SpatialDatasetIdentifierCode,
		}

		// Map the links
		log.Printf("Start mapping the Links...")
		for _, srcLink := range srcDatasetFeed.Links {
			dstDataset.Links = append(dstDataset.Links, OtherLink{
				Type:        srcLink.Title,
				URI:         srcLink.Href,
				ContentType: &srcLink.Type,
				Language:    &srcLink.Hreflang,
			})
		}
		log.Printf("Done mapping the Links...")

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
		log.Printf("Start mapping the Entries...")
		for _, srcEntry := range srcDatasetFeed.Entries {
			dstDownload := Download{
				Name:    srcEntry.TechnicalName,
				Content: &srcEntry.Content,
				Title:   &srcEntry.Title,
			}

			if srcEntry.Updated != nil {
				updatedString := srcEntry.Updated.Format(time.RFC3339)
				dstDownload.Updated = &updatedString
			}

			if srcEntry.SRS != nil {
				dstDownload.Srs = Srs{
					URI:  srcEntry.SRS.URI,
					Code: srcEntry.SRS.Name,
				}
			}

			// Map the links
			log.Printf("Start mapping the DownloadLinks...")
			for _, srcDownloadLink := range srcEntry.DownloadLinks {

				dstLink := Link{
					BlobKey: &srcDownloadLink.Data,
					Rel:     &srcDownloadLink.Rel,
				}

				if srcDownloadLink.Time != nil {
					dstLink.Updated = srcDownloadLink.Time
				}
				if srcDownloadLink.Version != nil {
					dstLink.Version = srcDownloadLink.Version
				}
				if srcDownloadLink.BBox != nil {
					dstLink.Bbox = &Bbox{
						Minx: GetStringAsFloat32(srcDownloadLink.BBox.MinX),
						Miny: GetStringAsFloat32(srcDownloadLink.BBox.MinY),
						Maxx: GetStringAsFloat32(srcDownloadLink.BBox.MaxX),
						Maxy: GetStringAsFloat32(srcDownloadLink.BBox.MaxY),
					}
				}
			}

			log.Printf("Done mapping the DownloadLinks...")
			dstDataset.Downloads = append(dstDataset.Downloads, dstDownload)
		}
		log.Printf("Done mapping the Entries...")
		dst.Spec.Service.Datasets = append(dst.Spec.Service.Datasets, dstDataset)
	}
	log.Printf("Start mapping the Datasets...")

	// Kubernetes
	log.Printf("Start mapping the Kubernetes Specs...")
	dst.Spec.Kubernetes = &Kubernetes{
		Lifecycle: &Lifecycle{},
	}
	if src.Spec.Lifecycle.TTLInDays != nil {
		dst.Spec.Kubernetes.Lifecycle.TTLInDays = GetIntPointer(int(*src.Spec.Lifecycle.TTLInDays))
	}
	log.Printf("Done mapping the Kubernetes Specs...")

	return nil
}

func createBaseURL(host string, general General) (baseURL string) {

	atomURI := fmt.Sprintf("%s/%s", general.DatasetOwner, general.Dataset)
	if general.Theme != nil {
		atomURI += fmt.Sprintf("/%s", *general.Theme)
	}
	atomURI += "/atom"

	if general.ServiceVersion != nil {
		atomURI += fmt.Sprintf("/%s", *general.ServiceVersion)
	}

	baseURL = fmt.Sprintf("%s/%s/index.xml", host, atomURI)
	return
}

func GetInt32Pointer(value int32) *int32 {
	return &value
}

func GetIntPointer(value int) *int {
	return &value
}

func GetFloat32AsString(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', 0, 32)
}

func GetStringAsFloat32(value string) float32 {
	float, _ := strconv.ParseFloat(value, 32)
	return float32(float)
}
