package atom_generator

import (
	"errors"
	"strings"
	"time"

	"github.com/cbroglie/mustache"
	atom_feed "github.com/pdok/atom-generator/feeds"
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	v1 "github.com/pdok/smooth-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MapAtomV3ToAtomGeneratorConfig(atom pdoknlv3.Atom, ownerInfo v1.OwnerInfo) (atomGeneratorConfig atom_feed.Feeds, err error) {

	var describedbyLink, searchLink, relatedLink atom_feed.Link

	language := "nl"
	xmlSheet := pdoknlv3.GetBaseURL() + "/atom/style/style.xsl"
	selfLink := getSelfLink(atom, language)
	describedbyLink, err = getCSWDescribedbyLink(atom, language, ownerInfo)
	if err != nil {
		return atom_feed.Feeds{}, err
	}
	searchLink, err = getSearchLink(atom, language, ownerInfo)
	if err != nil {
		return atom_feed.Feeds{}, err
	}
	relatedLink, err = getHTMLRelatedLink(atom, language, ownerInfo)
	if err != nil {
		return atom_feed.Feeds{}, err
	}
	latestUpdated, err := getLatestUpdate(atom.Spec.DatasetFeeds)
	if err != nil {
		return atom_feed.Feeds{}, err
	}

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
				Author:  getAuthor(ownerInfo.Spec.Atom.Author),
				Entry:   getEntriesArray(atom),
			},
		},
	}
	return atomGeneratorConfig, err
}

func getLatestUpdate(feeds []pdoknlv3.DatasetFeed) (string, error) {
	if len(feeds) == 0 {
		return "", errors.New("Atom heeft geen dataset feeds.")
	}

	var updateTime *metav1.Time
	for _, datasetFeed := range feeds {
		for _, entry := range datasetFeed.Entries {
			if entry.Updated != nil && (updateTime == nil || updateTime.Before(entry.Updated)) {
				updateTime = entry.Updated
			}
		}
	}
	if updateTime == nil {
		return "", nil
	}
	return updateTime.Format(time.RFC3339), nil
}

func getEntriesArray(atom pdoknlv3.Atom) []atom_feed.Entry {
	var retEntriesArray []atom_feed.Entry
	for _, datasetFeed := range atom.Spec.DatasetFeeds {
		for _, entry := range datasetFeed.Entries {

			singleEntry := atom_feed.Entry{
				ID:                                entry.TechnicalName,
				Title:                             entry.Title,
				Content:                           entry.Content,
				Summary:                           datasetFeed.Subtitle,
				Rights:                            atom.Spec.Service.Rights,
				SpatialDatasetIdentifierCode:      datasetFeed.SpatialDatasetIdentifierCode,
				SpatialDatasetIdentifierNamespace: datasetFeed.SpatialDatasetIdentifierNamespace,
				Link:                              getEntryLinksArray(entry),
			}
			if entry.Updated != nil {
				updateTime := entry.Updated.Format(time.RFC3339)
				singleEntry.Updated = &updateTime
			}
			if entry.SRS != nil {
				singleEntry.Category = getCategory(entry.SRS)
			}
			if entry.Polygon != nil {
				singleEntry.Polygon = getBoundingBoxPolygon(entry.Polygon.BBox)
			}

			retEntriesArray = append(retEntriesArray, singleEntry)
		}
	}

	return retEntriesArray
}

func getEntryLinksArray(entry pdoknlv3.Entry) []atom_feed.Link {
	linksArray := []atom_feed.Link{}
	for _, link := range entry.DownloadLinks {
		dataLink := pdoknlv3.GetBlobEndpoint() + "/" + link.Data

		l := atom_feed.Link{
			Data:    &dataLink,
			Rel:     link.Rel,
			Version: link.Version,
			Time:    link.Time,
		}

		if link.BBox != nil {
			bboxString := getBboxString(link.BBox)
			l.Bbox = &bboxString
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
	sb.WriteString(bbox.MinX + " " + bbox.MinY)
	return sb.String()
}

func getAuthor(author v1.Author) atom_feed.Author {
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

func replaceMustachTemplate(hrefTemplate string, identifier string) (string, error) {
	templateVariable := map[string]string{"identifier": identifier}
	return mustache.Render(hrefTemplate, templateVariable)
}

func getCSWDescribedbyLink(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo) (atom_feed.Link, error) {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "csw" {
			href, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
			if err != nil {
				return atom_feed.Link{}, err
			}
			return atom_feed.Link{
				Rel:      "describedby",
				Href:     href,
				Type:     "application/xml",
				Hreflang: &language,
			}, nil
		}
	}
	return atom_feed.Link{}, errors.New("OwnerInfo heeft geen CSW template")
}

func getSearchLink(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo) (atom_feed.Link, error) {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "opensearch" {
			href, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.OpenSearch.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
			if err != nil {
				return atom_feed.Link{}, err
			}

			return atom_feed.Link{
				Rel:      "search",
				Href:     href,
				Type:     "application/xml",
				Hreflang: &language,
			}, nil
		}
	}
	return atom_feed.Link{}, errors.New("OwnerInfo heeft geen opensearch template")
}

func getHTMLRelatedLink(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo) (atom_feed.Link, error) {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "html" {
			href, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.HTML.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
			if err != nil {
				return atom_feed.Link{}, err
			}
			return atom_feed.Link{
				Rel:      "related",
				Href:     href,
				Type:     "text/html",
				Hreflang: &language,
			}, nil
		}
	}
	return atom_feed.Link{}, errors.New("OwnerInfo heeft geen html template")
}
