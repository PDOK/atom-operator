package atom_generator

import (
	atom_feed "github.com/pdok/atom-generator/feeds"
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
)

func MapAtomV3ToAtomGeneratorConfig(atom pdoknlv3.Atom) (atomGeneratorConfig atom_feed.Feeds, err error) {

	lang := "nl"

	atomGeneratorConfig = atom_feed.Feeds{
		Feeds: []atom_feed.Feed{
			{
				ID:         atom.Spec.Service.BaseURL + "/index.xml",
				InspireDls: "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0",
				Lang:       &lang,
				// XMLStylesheet: "",
			},
		},
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
	return atomGeneratorConfig, err
}
