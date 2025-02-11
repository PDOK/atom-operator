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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

// ConvertTo converts this Atom (v2beta1) to the Hub version (v3).
func (src *Atom) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*pdoknlv3.Atom)
	log.Printf("ConvertTo: Converting Atom from Spoke version v2beta1 to Hub version v3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v2beta1 to v3

	// Lifecycle
	dst.Spec.Lifecycle.TTLInDays = GetInt32Pointer(int32(*src.Spec.Kubernetes.Lifecycle.TTLInDays))

	// Service
	dst.Spec.Service = pdoknlv3.Service{
		// Todo BaseURL opbouwen
		BaseURL:    "http://localhost/owner/dataset",
		Lang:       "nl",
		Stylesheet: "https://service.pdok.nl/atom/style/style.xsl",
		Title:      src.Spec.Service.Title,
		Subtitle:   src.Spec.Service.Subtitle,
		// Todo metadata-id invullen in links
		Links: []pdoknlv3.Link{
			{
				Href:     "https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=<id>",
				Category: "metadataXml",
			},
			{
				Href:     "https://www.ngr.nl/geonetwork/opensearch/dut/<id>>/OpenSearchDescription.xml",
				Category: "opensearch",
			},
			{
				Href:     "https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/<id>",
				Category: "metadataHtml",
				Rel:      "describedBy",
			},
		},
		Rights: src.Spec.Service.Rights,
		Author: pdoknlv3.Author{
			Name:  "PDOK Beheer",
			Email: "beheerPDOK@kadaster.nl",
		},
	}

	dst.Spec.DatasetFeeds = []pdoknlv3.DatasetFeed{}
	for _, srcDataset := range src.Spec.Service.Datasets {
		dstDatasetFeed := pdoknlv3.DatasetFeed{
			TechnicalName: "<id>.xml",
			Title:         srcDataset.Title,
			Subtitle:      srcDataset.Subtitle,
			Author: pdoknlv3.Author{
				Name:  "",
				Email: "",
			},
			SpatialDatasetIdentifierCode:      srcDataset.SourceIdentifier,
			SpatialDatasetIdentifierNamespace: "",
		}

		// Map the links
		for _, srcLink := range srcDataset.Links {
			dstDatasetFeed.Links = append(dstDatasetFeed.Links, pdoknlv3.Link{
				Title:    srcLink.Type,
				Href:     srcLink.URI,
				Type:     *srcLink.ContentType,
				Hreflang: *srcLink.Language,
			})
		}

		// Map the entries
		for _, srcDownload := range srcDataset.Downloads {
			dstEntry := pdoknlv3.Entry{
				TechnicalName: srcDownload.Name, // TechnicalName vs Name?
				Title:         *srcDownload.Title,
				Content:       *srcDownload.Content,
				Updated:       nil, //TODO Convert from srcDownload.Updated
				// TODO fix polygon float dangerousTypes
				//Polygon: pdoknlv3.Polygon{
				//	BBox: pdoknlv3.BBox{
				//		MinX: strconv.FormatFloat(srcDataset.Bbox.Minx, 'f', -1, 32),
				//		MinY: strconv.FormatFloat(srcDataset.Bbox.Miny, 'f', -1, 32),
				//		MaxX: strconv.FormatFloat(srcDataset.Bbox.Maxx, 'f', -1, 32),
				//		MaxY: strconv.FormatFloat(srcDataset.Bbox.Maxy, 'f', -1, 32),
				//	},
				//},
				SRS: &pdoknlv3.SRS{
					URI:  srcDownload.Srs.URI,
					Name: srcDownload.Srs.Code,
				},
			}

			// Map the links
			for _, srcLink := range srcDownload.Links {
				dstEntry.DownloadLinks = append(dstEntry.DownloadLinks, pdoknlv3.DownloadLink{
					Data:    *srcLink.BlobKey,
					Time:    srcLink.Updated,
					Rel:     *srcLink.Rel,
					Version: srcLink.Version,
					// Todo bbox
				})
			}
			dstDatasetFeed.Entries = append(dstDatasetFeed.Entries, dstEntry)
		}

		dst.Spec.DatasetFeeds = append(dst.Spec.DatasetFeeds, dstDatasetFeed)
	}

	return nil
}

// ConvertFrom converts the Hub version (v3) to this Atom (v2beta1).
func (dst *Atom) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*pdoknlv3.Atom)
	log.Printf("ConvertFrom: Converting Atom from Hub version v3 to Spoke version v2beta1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v3 to v2beta1

	// General
	dst.Spec.General = General{ // Todo waar halen we deze info vandaan
		Dataset:        "",
		DatasetOwner:   "",
		DataVersion:    new(string),
		ServiceVersion: new(string),
		Theme:          new(string),
	}

	// Service
	dst.Spec.Service = AtomService{
		Title:    src.Spec.Service.Title,
		Subtitle: src.Spec.Service.Subtitle,
		//MetadataIdentifier: Todo take from service.links?
		Rights: src.Spec.Service.Rights,
		Author: Author{
			Name:  src.Spec.Service.Author.Name,
			Email: src.Spec.Service.Author.Email,
		},
	}

	// Datasets
	dst.Spec.Service.Datasets = []Dataset{}
	for _, srcDatasetFeed := range src.Spec.DatasetFeeds {
		dstDataset := Dataset{
			Name:               srcDatasetFeed.TechnicalName,
			Title:              srcDatasetFeed.Title,
			Subtitle:           srcDatasetFeed.Subtitle,
			MetadataIdentifier: "", // Todo take from Links?
			SourceIdentifier:   srcDatasetFeed.SpatialDatasetIdentifierCode,
		}

		// Map the links
		for _, srcLink := range srcDatasetFeed.Links {
			dstDataset.Links = append(dstDataset.Links, OtherLink{
				Type:        srcLink.Title,
				URI:         srcLink.Href,
				ContentType: &srcLink.Type,
				Language:    &srcLink.Hreflang,
			})
		}

		// TODO Bbox

		// Map the downloads
		for _, srcEntry := range srcDatasetFeed.Entries {
			dstDownload := Download{
				Name:    srcEntry.TechnicalName,
				Updated: nil,
				Content: &srcEntry.Content,
				Title:   &srcEntry.Title,
				// Todo links
				Srs: Srs{
					URI:  srcEntry.SRS.URI,
					Code: srcEntry.SRS.Name,
				},
			}
			// Map the links
			for _, srcDownloadLink := range srcEntry.DownloadLinks {
				dstDownload.Links = append(dstDownload.Links, Link{
					BlobKey: &srcDownloadLink.Data,
					Updated: srcDownloadLink.Time,
					Rel:     &srcDownloadLink.Rel,
					Version: srcDownloadLink.Version,
					// Todo bbox
				})
			}
			dstDataset.Downloads = append(dstDataset.Downloads, dstDownload)
		}

		dst.Spec.Service.Datasets = append(dst.Spec.Service.Datasets, dstDataset)
	}

	// Kubernetes
	dst.Spec.Kubernetes = &Kubernetes{
		Lifecycle: &Lifecycle{
			TTLInDays: GetIntPointer(int(*src.Spec.Lifecycle.TTLInDays)),
		},
	}

	return nil
}

func GetInt32Pointer(value int32) *int32 {
	return &value
}

func GetIntPointer(value int) *int {
	return &value
}
