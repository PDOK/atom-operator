package generator

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/cbroglie/mustache"
	atomfeed "github.com/pdok/atom-generator/feeds"
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	v1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MapAtomV3ToAtomGeneratorConfig(atom pdoknlv3.Atom, ownerInfo v1.OwnerInfo) (atomGeneratorConfig atomfeed.Feeds, err error) {

	var describedbyLink, searchLink, relatedLink atomfeed.Link

	language := "nl"
	xmlSheet := pdoknlv3.GetBaseURL() + "/atom/style/style.xsl"
	selfLink := getSelfLink(atom, language)
	describedbyLink, err = getCSWDescribedbyLink(atom, language, ownerInfo)
	if err != nil {
		return atomfeed.Feeds{}, err
	}
	searchLink, err = getSearchLink(atom, language, ownerInfo)
	if err != nil {
		return atomfeed.Feeds{}, err
	}
	relatedLink, err = getHTMLRelatedLink(atom, language, ownerInfo)
	if err != nil {
		return atomfeed.Feeds{}, err
	}
	latestUpdated, err := getLatestUpdate(atom.Spec.DatasetFeeds)
	if err != nil {
		return atomfeed.Feeds{}, err
	}

	atomGeneratorConfig.Feeds = []atomfeed.Feed{}
	serviceFeed := atomfeed.Feed{
		XMLStylesheet: &xmlSheet,
		Xmlns:         "http://www.w3.org/2005/Atom",
		Georss:        "http://www.georss.org/georss",
		InspireDls:    "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0",
		Lang:          &language,
		ID:            atom.Spec.Service.BaseURL + "/index.xml",
		Title:         escapeQuotes(atom.Spec.Service.Title),
		Subtitle:      escapeQuotes(atom.Spec.Service.Subtitle),
		// Feed Links
		Link: []atomfeed.Link{
			selfLink,
			describedbyLink,
			searchLink,
			relatedLink,
		},
		Rights:  atom.Spec.Service.Rights,
		Updated: &latestUpdated,
		Author:  getServiceAuthor(ownerInfo.Spec.Atom.Author),
		Entry:   getServiceEntries(atom, language, ownerInfo, &latestUpdated),
	}
	atomGeneratorConfig.Feeds = append(atomGeneratorConfig.Feeds, serviceFeed)

	for _, datasetFeed := range atom.Spec.DatasetFeeds {
		dsFeed := atomfeed.Feed{
			ID:            atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + ".xml",
			Title:         escapeQuotes(datasetFeed.Title),
			Subtitle:      escapeQuotes(datasetFeed.Subtitle),
			Lang:          &language,
			Link:          getDatasetLinks(atom, ownerInfo, datasetFeed),
			Rights:        atom.Spec.Service.Rights,
			XMLStylesheet: &xmlSheet,
			Author:        getDatasetAuthor(datasetFeed.Author),
			Entry:         getDatasetEntries(atom, datasetFeed),
		}
		atomGeneratorConfig.Feeds = append(atomGeneratorConfig.Feeds, dsFeed)
	}
	return atomGeneratorConfig, err
}

func getLatestUpdate(feeds []pdoknlv3.DatasetFeed) (string, error) {
	if len(feeds) == 0 {
		return "", errors.New("this atom doesn't have any dataset feeds")
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

func getServiceEntries(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo, latestUpdated *string) []atomfeed.Entry {
	var retEntriesArray []atomfeed.Entry
	for _, datasetFeed := range atom.Spec.DatasetFeeds {
		datasetEntry := atomfeed.Entry{
			ID:                                atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + ".xml",
			Title:                             escapeQuotes(datasetFeed.Title),
			SpatialDatasetIdentifierCode:      datasetFeed.SpatialDatasetIdentifierCode,
			SpatialDatasetIdentifierNamespace: datasetFeed.SpatialDatasetIdentifierNamespace,
			Link:                              getServiceEntryLinks(atom, language, ownerInfo, datasetFeed),
			Updated:                           latestUpdated,
			Summary:                           escapeQuotes(datasetFeed.Subtitle),
			Category:                          []atomfeed.Category{},
		}
		// Take the polygon bbox of the first entry, assuming all are equal
		if len(datasetFeed.Entries) > 0 {
			datasetEntry.Polygon = datasetFeed.Entries[0].Polygon.BBox.ToPolygon()
		}

		// Collect all categories
		for _, entry := range datasetFeed.Entries {
			if entry.SRS != nil {
				category := getCategory(entry.SRS)
				// Add category to datasetFeed.category if not yet present
				if !slices.Contains(datasetEntry.Category, category) {
					datasetEntry.Category = append(datasetEntry.Category, category)
				}
			}
		}

		retEntriesArray = append(retEntriesArray, datasetEntry)
	}

	return retEntriesArray
}

func getServiceEntryLinks(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo, datasetFeed pdoknlv3.DatasetFeed) []atomfeed.Link {
	describedByLinkHref, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
	describedByLink := atomfeed.Link{
		Rel:      "describedby",
		Href:     describedByLinkHref,
		Type:     "application/xml",
		Hreflang: &language,
	}

	alternateLink := atomfeed.Link{
		Rel:   "alternate",
		Href:  atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + ".xml",
		Type:  "application/atom+xml",
		Title: datasetFeed.Title,
	}
	return []atomfeed.Link{
		describedByLink,
		alternateLink,
	}

}

func getCategory(srs *pdoknlv3.SRS) atomfeed.Category {
	return atomfeed.Category{
		Term:  srs.URI,
		Label: srs.Name,
	}
}

func getServiceAuthor(author smoothoperatormodel.Author) atomfeed.Author {
	return atomfeed.Author{
		Name:  author.Name,
		Email: author.Email,
	}
}

func getDatasetAuthor(author smoothoperatormodel.Author) atomfeed.Author {
	return atomfeed.Author{
		Name:  author.Name,
		Email: author.Email,
	}
}

func getSelfLink(atom pdoknlv3.Atom, language string) atomfeed.Link {
	return atomfeed.Link{
		Rel:      "self",
		Href:     atom.Spec.Service.BaseURL + "/index.xml",
		Title:    escapeQuotes(atom.Spec.Service.Title),
		Type:     "application/atom+xml",
		Hreflang: &language,
	}
}

func replaceMustachTemplate(hrefTemplate string, identifier string) (string, error) {
	templateVariable := map[string]string{"identifier": identifier}
	return mustache.Render(hrefTemplate, templateVariable)
}

func getCSWDescribedbyLink(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo) (atomfeed.Link, error) {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "csw" {
			href, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
			if err != nil {
				return atomfeed.Link{}, err
			}
			return atomfeed.Link{
				Rel:      "describedby",
				Href:     href,
				Type:     "application/xml",
				Hreflang: &language,
			}, nil
		}
	}
	return atomfeed.Link{}, errors.New("ownerInfo doesn't have a CSW template")
}

func getSearchLink(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo) (atomfeed.Link, error) {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "opensearch" {
			href, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.OpenSearch.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
			if err != nil {
				return atomfeed.Link{}, err
			}

			return atomfeed.Link{
				Rel:      "search",
				Href:     href,
				Type:     "application/xml",
				Hreflang: &language,
			}, nil
		}
	}
	return atomfeed.Link{}, errors.New("ownerInfo doesn't have an opensearch template")
}

func getHTMLRelatedLink(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo) (atomfeed.Link, error) {
	for _, template := range atom.Spec.Service.ServiceMetadataLinks.Templates {
		if template == "html" {
			href, err := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.HTML.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
			if err != nil {
				return atomfeed.Link{}, err
			}
			return atomfeed.Link{
				Rel:      "related",
				Href:     href,
				Type:     "text/html",
				Hreflang: &language,
			}, nil
		}
	}
	return atomfeed.Link{}, errors.New("ownerInfo doesn't have a html template")
}

func getDatasetLinks(atom pdoknlv3.Atom, ownerInfo v1.OwnerInfo, datasetFeed pdoknlv3.DatasetFeed) []atomfeed.Link {

	selfLink := atomfeed.Link{
		Rel:  "self",
		Href: atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + ".xml",
	}
	upLink := atomfeed.Link{
		Rel:   "up",
		Href:  atom.Spec.Service.BaseURL + "/index.xml",
		Type:  "application/atom+xml",
		Title: "Top Atom Download Service Feed",
	}
	describedByLinkHref, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
	describedbyLink := atomfeed.Link{
		Rel:  "describedby",
		Href: describedByLinkHref,
		Type: "text.html",
	}
	relatedLinkHref, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.HTML.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
	relatedLink := atomfeed.Link{
		Href:  relatedLinkHref,
		Type:  "text.html",
		Title: "NGR pagina voor deze dataset",
	}

	links := []atomfeed.Link{
		selfLink,
		upLink,
		describedbyLink,
		relatedLink,
	}

	for _, link := range datasetFeed.Links {
		linkDescribedbyLink := atomfeed.Link{
			Rel:      "describedby",
			Href:     link.Href,
			Title:    escapeQuotes(link.Title),
			Type:     link.Type,
			Hreflang: &link.Hreflang,
		}
		links = append(links, linkDescribedbyLink)
	}

	return links
}

func getDatasetEntries(atom pdoknlv3.Atom, datasetFeed pdoknlv3.DatasetFeed) []atomfeed.Entry {
	var entries []atomfeed.Entry
	for _, entry := range datasetFeed.Entries {

		datasetEntry := atomfeed.Entry{
			ID:       atom.Spec.Service.BaseURL + "/" + entry.TechnicalName + ".xml",
			Title:    escapeQuotes(entry.Title),
			Content:  entry.Content,
			Link:     []atomfeed.Link{},
			Rights:   atom.Spec.Service.Rights,
			Category: []atomfeed.Category{getCategory(entry.SRS)},
			Polygon:  entry.Polygon.BBox.ToPolygon(),
			Summary:  escapeQuotes(datasetFeed.Subtitle),
		}

		updated := entry.Updated.Format(time.RFC3339)
		datasetEntry.Updated = &updated

		emptyRelCount := getEmptyRelCount(entry)
		for _, downloadLink := range entry.DownloadLinks {
			link := atomfeed.Link{
				Rel:   getDownloadLinkRel(downloadLink, emptyRelCount),
				Href:  getDownloadLinkHref(downloadLink, atom),
				Data:  getDownloadLinkData(downloadLink),
				Title: getDownloadLinkTitle(datasetFeed, entry, downloadLink),
			}

			if downloadLink.Version != nil {
				link.Version = downloadLink.Version
			}
			if downloadLink.Time != nil {
				link.Time = downloadLink.Time
			}
			if downloadLink.BBox != nil {
				bboxString := downloadLink.BBox.ToExtent()
				link.Bbox = &bboxString
			}

			datasetEntry.Link = append(datasetEntry.Link, link)
		}
		entries = append(entries, datasetEntry)
	}

	return entries
}

func getEmptyRelCount(entry pdoknlv3.Entry) (count int) {
	for _, downloadLink := range entry.DownloadLinks {
		if downloadLink.Rel == "" {
			count++
		}
	}
	return
}

func getDownloadLinkRel(downloadLink pdoknlv3.DownloadLink, emptyRelCount int) (rel string) {
	switch {
	case downloadLink.Rel != "":
		rel = downloadLink.Rel
	case emptyRelCount > 1:
		rel = "section"
	default:
		rel = "alternate"
	}
	return
}

func getDownloadLinkHref(downloadLink pdoknlv3.DownloadLink, atom pdoknlv3.Atom) (href string) {
	href = atom.Spec.Service.BaseURL + "/downloads"
	if downloadLink.Version != nil {
		href += "/" + *downloadLink.Version
	}
	href += "/" + downloadLink.GetBlobName()
	return
}

// Using internal url, atom generator uses this url to determine content-length and
// content-type of the download and convert it into external url
func getDownloadLinkData(downloadLink pdoknlv3.DownloadLink) *string {
	data := pdoknlv3.GetBlobEndpoint() + "/" + downloadLink.Data
	return &data
}

func getDownloadLinkTitle(datasetFeed pdoknlv3.DatasetFeed, entry pdoknlv3.Entry, downloadLink pdoknlv3.DownloadLink) (title string) {
	if entry.Title != "" {
		title = entry.Title
	} else {
		title = datasetFeed.Title
	}
	title += "-"
	if downloadLink.Version != nil {
		title += *downloadLink.Version + " "
	}
	title += downloadLink.GetBlobName()
	return
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
