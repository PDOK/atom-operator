package atom_generator

import (
	atom_feed "github.com/pdok/atom-generator/feeds"
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	"strings"
	"time"
)

func MapAtomV3ToAtomGeneratorConfig(atom pdoknlv3.Atom) (atomGeneratorConfig atom_feed.Feeds, err error) {

	language := "nl"
	xmlSheet := "https://service.pdok.nl/atom/style/style.xsl"
	selfLink := getLinkByRelation(atom, language, "self")
	describedbyLink := getLinkByRelation(atom, language, "describedby")
	searchLink := getLinkByRelation(atom, language, "search")
	upLink := getLinkByRelation(atom, language, "up")
	relatedLink := getLinkByRelation(atom, language, "related")

	atomGeneratorConfig = atom_feed.Feeds{
		Feeds: []atom_feed.Feed{
			{
				//XMLName:       Name{"http://www.w3.org/2005/Atom", "feed"},
				XMLStylesheet: &xmlSheet,
				Xmlns:         "http://www.w3.org/2005/Atom",
				Georss:        "http://www.georss.org/georss",
				InspireDls:    "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0",
				Lang:          &language,
				ID:            atom.Spec.Service.BaseURL + "/index.xml",
				Title:         atom.Spec.Service.Title,
				Subtitle:      atom.Spec.Service.Subtitle,
				// Feed Links
				Self:        &selfLink,
				Describedby: &describedbyLink,
				Search:      &searchLink,
				Up:          &upLink,
				Link: []atom_feed.Link{
					relatedLink,
				},
				Rights: atom.Spec.Service.Rights,
				//Updated: atom.Spec.Service.U niet meer vindbaar
				Author: getAuthor(atom.Spec.Author),
				Entry:  getEntriesArray(atom),
			},
		},
	}
	return atomGeneratorConfig, err
}

func getEntriesArray(atom pdoknlv3.Atom) []atom_feed.Entry {
	var retEntriesArray []atom_feed.Entry
	for _, datasetFeed := range atom.Spec.DatasetFeeds {
		for _, entry := range datasetFeed.Entries {
			updateTime := entry.Updated.Format(time.RFC3339)

			singleEntry := atom_feed.Entry{
				ID:      entry.TechnicalName,
				Title:   entry.Title,
				Content: entry.Content,
				//Summary: entry.,
				//Rights: entry.Right,
				Updated: &updateTime,
				Polygon: getBoundingBoxPolygon(entry.Polygon),
				//SpatialDatasetIdentifierCode:      nil,
				//SpatialDatasetIdentifierNamespace: nil,
				//Category:                          nil,
				//Link:                              nil, // []Links
			}
			retEntriesArray = append(retEntriesArray, singleEntry)
		}
	}

	return retEntriesArray
}

func getBoundingBoxPolygon(polygon *pdoknlv3.Polygon) string {
	var sb strings.Builder
	// punt links beneden start van een polygon
	sb.WriteString(polygon.BBox.MinX + " " + polygon.BBox.MinY + " ")
	// punt links boven start van een polygon
	sb.WriteString(polygon.BBox.MinX + " " + polygon.BBox.MaxY + " ")
	// punt rechts boven start van een polygon
	sb.WriteString(polygon.BBox.MaxX + " " + polygon.BBox.MaxY + " ")
	// punt rechts beneden start van een polygon
	sb.WriteString(polygon.BBox.MaxX + " " + polygon.BBox.MinY + " ")
	// punt links beneden. eninde van een polygon is gelijk aan de start
	sb.WriteString(polygon.BBox.MinX + " " + polygon.BBox.MinY + " ")
	return sb.String()
}

func getLinkByRelation(atom pdoknlv3.Atom, language string, relation string) atom_feed.Link {
	for _, link := range atom.Spec.Service.Links {
		if link.Rel == relation {
			return atom_feed.Link{
				Rel:      relation,
				Href:     link.Href,
				Type:     link.Type,
				Title:    link.Title,
				Hreflang: &language,
			}
		}
	}
	return atom_feed.Link{}
}

func getAuthor(author pdoknlv3.Author) atom_feed.Author {
	return atom_feed.Author{
		Name:  author.Name,
		Email: author.Email,
	}
}

// atomGeneratorConfig = AtomGeneratorConfig{
//	Feeds: []Feed{
//		{
//			ID:         atom.Spec.Service.BaseURL + "/index.xml",
//			InspireDLS: "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0",
//			Lang:       "nl",
//			Stylesheet: "example.com/styles/atom.xsl",
//			Title:      "Service Title",
//			Subtitle:   "Service Subtitle",
//			Link: []Link{
//				{
//					Rel:   "self",
//					Href:  atom.Spec.Service.BaseURL + "/index.xml",
//					Title: "Service Title",
//					Type:  "application/atom+xml",
//				},
//				{
//					Rel:  "describedby",
//					Href: "example.com/getrecord?id=service1",
//					Type: "application/xml",
//				},
//				{
//					Rel:   "search",
//					Href:  "example.com/opensearch.xml",
//					Title: "Open Search document voor INSPIRE Download service PDOK",
//					Type:  "application/opensearchdescription+xml",
//				},
//				{
//					Rel:   "related",
//					Href:  "example.com/metadata/service1",
//					Type:  "text/html",
//					Title: "NGR pagina voor deze download service",
//				},
//			},
//			Rights:  "All rights reserved",
//			Updated: &updatedTime,
//			Author: Author{
//				Name:  "PDOK Beheer",
//				Email: "beheerPDOK@kadaster.nl",
//			},
//			Entry: []Entry{
//				{
//					ID:                                "example.com/atom/dataset1.xml",
//					Title:                             "Dataset 1 Title",
//					SpatialDatasetIdentifierCode:      "dataset1-id",
//					SpatialDatasetIdentifierNamespace: "http://www.pdok.nl",
//					Link: []Link{
//						{
//							Rel:  "describedby",
//							Href: "example.com/getrecord?id=dataset1",
//							Type: "application/xml",
//						},
//						{
//							Rel:   "alternate",
//							Href:  "example.com/atom/dataset1.xml",
//							Type:  "application/atom+xml",
//							Title: "Dataset 1 Title",
//						},
//					},
//					Updated: &updatedTime,
//					Summary: "Dataset 1 Subtitle",
//					Polygon: "42.0 12.0 42.0 13.0 43.0 13.0 43.0 12.0 42.0 12.0",
//					Category: []Category{
//						{
//							Term:  "urn:ogc:def:crs:EPSG::4326",
//							Label: "EPSG:4326",
//						},
//					},
//				},
//				// Adding another entry for completeness
//				{
//					ID:    "example.com/atom/dataset2.xml",
//					Title: "Dataset 2 Title",
//					Link: []Link{
//						{
//							Rel:  "self",
//							Href: "example.com/atom/dataset2.xml",
//						},
//						{
//							Rel:   "up",
//							Href:  "example.com/atom/index.xml",
//							Type:  "application/atom+xml",
//							Title: "Top Atom Download Service Feed",
//						},
//						{
//							Rel:  "describedby",
//							Href: "example.com/getrecord?id=service1",
//							Type: "text/html",
//						},
//						{
//							Rel:   "related",
//							Href:  "example.com/metadata/dataset2",
//							Type:  "text/html",
//							Title: "NGR pagina voor deze dataset",
//						},
//						{
//							Rel:      "describedby",
//							Href:     "example.com/link1",
//							Title:    "Link Type 1",
//							Type:     "application/pdf",
//							Hreflang: "en",
//						},
//						// Add more links if needed
//					},
//					Rights:  "All rights reserved",
//					Updated: "20-04-2024 huplelepup",
//				},
//			},
//		},
//	},
// }

// return

//func getSelfLink(atom pdoknlv3.Atom, language string) atom_feed.Link {
//	return atom_feed.Link{
//		Rel:      "self",
//		Href:     atom.Spec.Service.BaseURL + "/index.xml",
//		Title:    atom.Spec.Service.Title,
//		Type:     "application/atom+xml",
//		Hreflang: &language,
//	}
//}
//
//func getDescribedbyLink(atom pdoknlv3.Atom, language string) atom_feed.Link {
//	href, title := "", ""
//	for _, link := range atom.Spec.Service.Links {
//		if link.Rel == "describedby" {
//			href = link.Href
//			title = link.Title
//		}
//	}
//	return atom_feed.Link{
//		Rel:      "describedby",
//		Href:     href,
//		Type:     "application/xml",
//		Hreflang: &language,
//	}
//}
//
//func getSearchLink(atom pdoknlv3.Atom, language string) atom_feed.Link {
//	href, title := "", ""
//	for _, link := range atom.Spec.Service.Links {
//		if link.Rel == "search" {
//			href = link.Href
//			title = link.Title
//		}
//	}
//	return atom_feed.Link{
//		Rel:      "search",
//		Href:     href,
//		Type:     "application/opensearchdescription+xml",
//		Title:    title,
//		Hreflang: &language,
//	}
//}
