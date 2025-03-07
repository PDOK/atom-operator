package atom_generator

import (
	"errors"
	"slices"
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

	atomGeneratorConfig.Feeds = []atom_feed.Feed{}
	indexFeed := atom_feed.Feed{
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
		//Self:        &selfLink,
		//Describedby: &describedbyLink,
		//Search: &searchLink,
		Link: []atom_feed.Link{
			selfLink,
			describedbyLink,
			searchLink,
			relatedLink,
		},
		Rights:  atom.Spec.Service.Rights,
		Updated: &latestUpdated,
		Author:  getIndexAuthor(ownerInfo.Spec.Atom.Author),
		Entry:   getIndexEntries(atom, language, ownerInfo),
	}
	atomGeneratorConfig.Feeds = append(atomGeneratorConfig.Feeds, indexFeed)

	for _, datasetFeed := range atom.Spec.DatasetFeeds {

		dsFeed := atom_feed.Feed{
			ID:            atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + ".xml",
			Title:         datasetFeed.Title,
			Subtitle:      datasetFeed.Subtitle,
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

func getIndexEntries(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo) []atom_feed.Entry {
	var retEntriesArray []atom_feed.Entry
	for _, datasetFeed := range atom.Spec.DatasetFeeds {
		datasetEntry := atom_feed.Entry{
			ID:                                atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + "/index.xml",
			Title:                             datasetFeed.Title,
			SpatialDatasetIdentifierCode:      datasetFeed.SpatialDatasetIdentifierCode,
			SpatialDatasetIdentifierNamespace: datasetFeed.SpatialDatasetIdentifierNamespace,
			Link:                              getIndexEntryLinks(atom, language, ownerInfo, datasetFeed),
			Summary:                           datasetFeed.Subtitle,
			Category:                          []atom_feed.Category{},
		}
		// Take the polygon bbox of the first entry, assuming all are equal
		if datasetFeed.Entries != nil && len(datasetFeed.Entries) > 0 {
			datasetEntry.Polygon = getBoundingBoxPolygon(datasetFeed.Entries[0].Polygon.BBox)
		}

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
				category := getCategory(entry.SRS)
				singleEntry.Category = []atom_feed.Category{category}

				// Add category to datasetFeed.category if not yet present
				if slices.Contains(datasetEntry.Category, category) == false {
					datasetEntry.Category = append(datasetEntry.Category, category)
				}
			}
			if entry.Polygon != nil {
				singleEntry.Polygon = getBoundingBoxPolygon(entry.Polygon.BBox)
			}

			retEntriesArray = append(retEntriesArray, singleEntry)
		}
	}

	return retEntriesArray
}

func getIndexEntryLinks(atom pdoknlv3.Atom, language string, ownerInfo v1.OwnerInfo, datasetFeed pdoknlv3.DatasetFeed) []atom_feed.Link {
	describedByLinkHref, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
	describedByLink := atom_feed.Link{
		Rel:      "describedby",
		Href:     describedByLinkHref,
		Type:     "application/xml",
		Hreflang: &language,
	}

	alternateLink := atom_feed.Link{
		Rel:   "alternate",
		Href:  atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + ".xml",
		Type:  "application/atom+xml",
		Title: datasetFeed.Title,
	}
	return []atom_feed.Link{
		describedByLink,
		alternateLink,
	}

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

func getCategory(srs *pdoknlv3.SRS) atom_feed.Category {
	return atom_feed.Category{
		Term:  srs.URI,
		Label: srs.Name,
	}
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

func getIndexAuthor(author v1.Author) atom_feed.Author {
	return atom_feed.Author{
		Name:  author.Name,
		Email: author.Email,
	}
}

func getDatasetAuthor(author pdoknlv3.Author) atom_feed.Author {
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

func getDatasetLinks(atom pdoknlv3.Atom, ownerInfo v1.OwnerInfo, datasetFeed pdoknlv3.DatasetFeed) []atom_feed.Link {

	selfLink := atom_feed.Link{
		Rel:  "self",
		Href: atom.Spec.Service.BaseURL + "/" + datasetFeed.TechnicalName + ".xml",
	}
	upLink := atom_feed.Link{
		Rel:   "up",
		Href:  atom.Spec.Service.BaseURL + "/index.xml",
		Type:  "application/atom+xml",
		Title: "Top Atom Download Service Feed",
	}
	describedByLinkHref, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
	describedbyLink := atom_feed.Link{
		Rel:  "describedby",
		Href: describedByLinkHref,
		Type: "text.html",
	}
	relatedLinkHref, _ := replaceMustachTemplate(ownerInfo.Spec.MetadataUrls.HTML.HrefTemplate, atom.Spec.Service.ServiceMetadataLinks.MetadataIdentifier)
	relatedLink := atom_feed.Link{
		Href:  relatedLinkHref,
		Type:  "text.html",
		Title: "NGR pagina voor deze dataset",
	}

	links := []atom_feed.Link{
		selfLink,
		upLink,
		describedbyLink,
		relatedLink,
	}

	for _, link := range datasetFeed.Links {
		linkDescribedbyLink := atom_feed.Link{
			Rel:      "describedby",
			Href:     link.Href,
			Title:    link.Title,
			Type:     link.Type,
			Hreflang: &link.Hreflang,
		}
		links = append(links, linkDescribedbyLink)
	}

	return links
}

func getDatasetEntries(atom pdoknlv3.Atom, datasetFeed pdoknlv3.DatasetFeed) []atom_feed.Entry {
	entries := []atom_feed.Entry{}
	for _, entry := range datasetFeed.Entries {

		datasetEntry := atom_feed.Entry{
			ID:       atom.Spec.Service.BaseURL + "/" + entry.TechnicalName + ".xml",
			Title:    entry.Title,
			Content:  entry.Content,
			Link:     []atom_feed.Link{},
			Rights:   atom.Spec.Service.Rights,
			Category: []atom_feed.Category{getCategory(entry.SRS)},
			Polygon:  getBoundingBoxPolygon(entry.Polygon.BBox),
		}

		updated := entry.Updated.Format(time.RFC3339)
		datasetEntry.Updated = &updated

		emptyRelCount := getEmptyRelCount(entry)
		for _, downloadLink := range entry.DownloadLinks {
			link := atom_feed.Link{
				Rel:   getDownloadLinkRel(downloadLink, emptyRelCount),
				Href:  getDownloadLinkHref(downloadLink),
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
				bboxString := getBboxString(downloadLink.BBox)
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
	if downloadLink.Rel != "" {
		rel = downloadLink.Rel
	} else if emptyRelCount > 0 {
		rel = "section"
	} else {
		rel = "alternate"
	}
	return
}

func getDownloadLinkHref(downloadLink pdoknlv3.DownloadLink) (href string) {
	href = pdoknlv3.GetBaseURL() + "/downloads"
	if downloadLink.Version != nil {
		href += "/" + *downloadLink.Version
	}
	href += "/" + downloadLink.GetBlobName()
	return
}

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
