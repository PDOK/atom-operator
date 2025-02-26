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
	selfLink := getSelfLink(atom, language)
	describedbyLink := getCSWDescribedbyLink(atom, language)
	searchLink := getSearchLink(atom, language)
	relatedLink := getHTMLRelatedLink(atom, language)
	latestUpdated := getLatestUpdate(atom.Spec.DatasetFeeds)

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
				Link: []atom_feed.Link{
					relatedLink,
				},
				Rights:  atom.Spec.Service.Rights,
				Updated: &latestUpdated,
				Author:  getAuthor(atom.Spec.Author),
				Entry:   getEntriesArray(atom),
			},
		},
	}
	return atomGeneratorConfig, err
}

func getLatestUpdate(feeds []pdoknlv3.DatasetFeed) string {
	updateTime := feeds[0].Entries[0].Updated
	for _, datasetFeed := range feeds {
		for _, entry := range datasetFeed.Entries {
			if updateTime.Before(entry.Updated) {
				updateTime = entry.Updated
			}
		}
	}
	return updateTime.Format(time.RFC3339)
}

func getEntriesArray(atom pdoknlv3.Atom) []atom_feed.Entry {
	var retEntriesArray []atom_feed.Entry
	for _, datasetFeed := range atom.Spec.DatasetFeeds {
		for _, entry := range datasetFeed.Entries {
			updateTime := entry.Updated.Format(time.RFC3339)

			singleEntry := atom_feed.Entry{
				ID:                                entry.TechnicalName,
				Title:                             entry.Title,
				Content:                           entry.Content,
				Summary:                           datasetFeed.Subtitle,
				Rights:                            atom.Spec.Service.Rights,
				Updated:                           &updateTime,
				Polygon:                           getBoundingBoxPolygon(entry.Polygon.BBox),
				SpatialDatasetIdentifierCode:      datasetFeed.SpatialDatasetIdentifierCode,
				SpatialDatasetIdentifierNamespace: datasetFeed.SpatialDatasetIdentifierNamespace,
				Category:                          getCategory(entry.SRS),
				Link:                              getEntryLinksArray(entry), // []Links
			}
			retEntriesArray = append(retEntriesArray, singleEntry)
		}
	}

	return retEntriesArray
}

func getEntryLinksArray(entry pdoknlv3.Entry) []atom_feed.Link {
	linksArray := []atom_feed.Link{}
	for _, link := range entry.DownloadLinks {
		dataLink := link.Data
		bboxString := getBboxString(link.BBox)

		l := atom_feed.Link{
			Data:    &dataLink,
			Rel:     link.Rel,
			Version: link.Version,
			Time:    link.Time,
			Bbox:    &bboxString,
		}
		linksArray = append(linksArray, l)
	}
	return linksArray
}

func getBboxString(bbox *pdoknlv3.BBox) string {
	var sb strings.Builder
	sb.WriteString(bbox.MinX + " " + bbox.MinY + " " + bbox.MaxX + " " + bbox.MaxY)
	return sb.String()
}

func getCategory(srs *pdoknlv3.SRS) []atom_feed.Category {
	cat := []atom_feed.Category{
		{
			Term:  srs.URI,
			Label: srs.Name,
		},
	}
	return cat
}

func getBoundingBoxPolygon(bbox pdoknlv3.BBox) string {
	var sb strings.Builder
	// punt links beneden start van een polygon
	sb.WriteString(bbox.MinX + " " + bbox.MinY + " ")
	// punt links boven start van een polygon
	sb.WriteString(bbox.MinX + " " + bbox.MaxY + " ")
	// punt rechts boven start van een polygon
	sb.WriteString(bbox.MaxX + " " + bbox.MaxY + " ")
	// punt rechts beneden start van een polygon
	sb.WriteString(bbox.MaxX + " " + bbox.MinY + " ")
	// punt links beneden. eninde van een polygon is gelijk aan de start
	sb.WriteString(bbox.MinX + " " + bbox.MinY + " ")
	return sb.String()
}

func getAuthor(author pdoknlv3.Author) atom_feed.Author {
	return atom_feed.Author{
		Name:  author.Name,
		Email: author.Email,
	}
}

func getSelfLink(atom pdoknlv3.Atom, language string) atom_feed.Link {
	return atom_feed.Link{
		Rel:      "self",
		Href:     atom.Spec.Service.BaseURL + "/index.xml",
		Title:    strings.Replace(atom.Spec.Service.Title, "\"", "\\\"", -1),
		Type:     "application/atom+xml",
		Hreflang: &language,
	}
}

// TODO: Maak gebruik van de MetadataUrls uit https://github.com/PDOK/operator-commons/blob/master/api/v1/ownerinfo_types.go  Voor nu is hardcoded urlś zijn gebouwd.
func getCSWDescribedbyLink(atom pdoknlv3.Atom, language string) atom_feed.Link {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "csw" {
			return atom_feed.Link{
				Rel:      "describedby",
				Href:     "https://www.nationaalgeoregister.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id=" + atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier,
				Type:     "application/xml",
				Hreflang: &language,
			}
		}
	}
	return atom_feed.Link{}
}

func getSearchLink(atom pdoknlv3.Atom, language string) atom_feed.Link {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "opensearch" {
			return atom_feed.Link{
				Rel:      "search",
				Href:     "https://www.nationaalgeoregister.nl/geonetwork/opensearch/dut/" + atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier + "/OpenSearchDescription.xml",
				Type:     "application/xml",
				Hreflang: &language,
			}
		}
	}
	return atom_feed.Link{}
}

func getHTMLRelatedLink(atom pdoknlv3.Atom, language string) atom_feed.Link {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "html" {
			return atom_feed.Link{
				Rel:      "related",
				Href:     "https://www.nationaalgeoregister.nl/geonetwork/srv/dut/catalog.search#/metadata/" + atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier,
				Type:     "text/html",
				Hreflang: &language,
			}
		}
	}
	return atom_feed.Link{}
}
