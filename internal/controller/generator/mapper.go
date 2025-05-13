package generator

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/cbroglie/mustache"
	atomfeed "github.com/pdok/atom-generator/feeds"
	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothoperatorv1 "github.com/pdok/smooth-operator/api/v1"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	smoothutil "github.com/pdok/smooth-operator/pkg/util"
)

func MapAtomV3ToAtomGeneratorConfig(atom pdoknlv3.Atom, ownerInfo smoothoperatorv1.OwnerInfo) (atomGeneratorConfig atomfeed.Feeds, err error) {
	if ownerInfo.Spec.Atom == nil {
		return atomGeneratorConfig, errors.New("ownerInfo has no Atom information defined")
	}

	selfLink := getSelfLink(atom)
	links := []atomfeed.Link{selfLink}
	if atom.Spec.Service.ServiceMetadataLinks != nil {
		err = addMetadataLinks(*atom.Spec.Service.ServiceMetadataLinks, ownerInfo, &links, "NGR pagina voor deze download service", false)
		if err != nil {
			return atomfeed.Feeds{}, err
		}
	}

	// TODO append custom links to links (requires mapping)
	// links = append(links, atom.Spec.Service.Links...)

	stylesheet := atom.Spec.Service.Stylesheet
	if atom.Spec.Service.Stylesheet == nil {
		stylesheet = ownerInfo.Spec.Atom.DefaultStylesheet
	}
	var xmlStylesheet *string
	if stylesheet != nil {
		xmlStylesheet = smoothutil.Pointer(stylesheet.String())
	}

	atomGeneratorConfig.Feeds = []atomfeed.Feed{}
	entries, err := getServiceEntries(atom, ownerInfo)
	if err != nil {
		return atomfeed.Feeds{}, err
	}
	serviceFeed := atomfeed.Feed{
		XMLStylesheet: xmlStylesheet,
		Xmlns:         "http://www.w3.org/2005/Atom",
		Georss:        "http://www.georss.org/georss",
		InspireDls:    "http://inspire.ec.europa.eu/schemas/inspire_dls/1.0",
		Lang:          &atom.Spec.Service.Lang,
		ID:            atom.Spec.Service.BaseURL.JoinPath("index.xml").String(),
		Title:         escapeQuotes(atom.Spec.Service.Title),
		Subtitle:      escapeQuotes(atom.Spec.Service.Subtitle),
		// Index Feed Links
		Link:   links,
		Rights: atom.Spec.Service.Rights,
		Author: getAuthor(ownerInfo.Spec.Atom.Author),
		Entry:  entries,
	}
	atomGeneratorConfig.Feeds = append(atomGeneratorConfig.Feeds, serviceFeed)

	for _, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		datasetLinks, err := getDatasetLinks(atom, ownerInfo, datasetFeed)
		if err != nil {
			return atomfeed.Feeds{}, err
		}
		dsFeed := atomfeed.Feed{
			ID:            atom.Spec.Service.BaseURL.JoinPath(datasetFeed.TechnicalName + ".xml").String(),
			Title:         escapeQuotes(datasetFeed.Title),
			Subtitle:      escapeQuotes(datasetFeed.Subtitle),
			Lang:          &atom.Spec.Service.Lang,
			Link:          datasetLinks,
			Rights:        atom.Spec.Service.Rights,
			XMLStylesheet: xmlStylesheet,
			Author:        getAuthor(datasetFeed.Author),
			Entry:         getDatasetEntries(atom, datasetFeed),
		}
		atomGeneratorConfig.Feeds = append(atomGeneratorConfig.Feeds, dsFeed)
	}
	return atomGeneratorConfig, err
}

func getServiceEntries(atom pdoknlv3.Atom, ownerInfo smoothoperatorv1.OwnerInfo) ([]atomfeed.Entry, error) {
	var retEntriesArray []atomfeed.Entry
	for _, datasetFeed := range atom.Spec.Service.DatasetFeeds {
		id := atom.Spec.Service.BaseURL.JoinPath(datasetFeed.TechnicalName + ".xml").String()
		var links []atomfeed.Link
		if datasetFeed.DatasetMetadataLinks != nil {
			err := addMetadataLinks(*datasetFeed.DatasetMetadataLinks, ownerInfo, &links, "", true)
			if err != nil {
				return nil, err
			}
		}
		alternateLink := atomfeed.Link{
			Rel:   "alternate",
			Href:  id,
			Type:  "application/atom+xml",
			Title: escapeQuotes(datasetFeed.Title),
		}
		links = append(links, alternateLink)
		datasetEntry := atomfeed.Entry{
			ID:                                id,
			Title:                             escapeQuotes(datasetFeed.Title),
			SpatialDatasetIdentifierCode:      datasetFeed.SpatialDatasetIdentifierCode,
			SpatialDatasetIdentifierNamespace: datasetFeed.SpatialDatasetIdentifierNamespace,
			Link:                              links,
			Summary:                           escapeQuotes(datasetFeed.Subtitle),
			Category:                          []atomfeed.Category{},
		}

		// TODO willen we hier een verbetering doorvoeren dat het altijd de max polygon van alle entries maakt?
		// Take the polygon bbox of the first entry, assuming all are equal
		if len(datasetFeed.Entries) > 0 {
			datasetEntry.Polygon = datasetFeed.Entries[0].Polygon.BBox.ToPolygon()
		}

		// Collect all categories
		for _, entry := range datasetFeed.Entries {
			category := getCategory(entry.SRS)
			// Add category to datasetFeed.category if not yet present
			if !slices.Contains(datasetEntry.Category, category) {
				datasetEntry.Category = append(datasetEntry.Category, category)
			}
		}

		retEntriesArray = append(retEntriesArray, datasetEntry)
	}

	return retEntriesArray, nil
}

func getCategory(srs pdoknlv3.SRS) atomfeed.Category {
	return atomfeed.Category{
		Term:  srs.URI.String(),
		Label: srs.Name,
	}
}

func getAuthor(author smoothoperatormodel.Author) atomfeed.Author {
	return atomfeed.Author{
		Name:  author.Name,
		Email: author.Email,
	}
}

func getSelfLink(atom pdoknlv3.Atom) atomfeed.Link {
	return atomfeed.Link{
		Rel:   "self",
		Href:  atom.Spec.Service.BaseURL.JoinPath("index.xml").String(),
		Title: escapeQuotes(atom.Spec.Service.Title),
		Type:  "application/atom+xml",
	}
}

func replaceMustacheTemplate(hrefTemplate string, identifier string) (string, error) {
	templateVariable := map[string]string{"identifier": identifier}
	return mustache.Render(hrefTemplate, templateVariable)
}

func addMetadataLinks(metadataLinks pdoknlv3.MetadataLink, ownerInfo smoothoperatorv1.OwnerInfo, links *[]atomfeed.Link, htmlTitle string, onlyCSW bool) error {
	for _, template := range metadataLinks.Templates {
		if template == "csw" {
			href, err := replaceMustacheTemplate(ownerInfo.Spec.MetadataUrls.CSW.HrefTemplate, metadataLinks.MetadataIdentifier)
			if err != nil {
				return err
			}
			link := atomfeed.Link{
				Rel:  "describedby",
				Href: href,
				Type: "application/xml",
			}
			*links = append(*links, link)
			if onlyCSW {
				return nil
			}
		}
		if template == "opensearch" {
			href, err := replaceMustacheTemplate(ownerInfo.Spec.MetadataUrls.OpenSearch.HrefTemplate, metadataLinks.MetadataIdentifier)
			if err != nil {
				return err
			}
			link := atomfeed.Link{
				Rel:   "search",
				Href:  href,
				Title: "Open Search document voor INSPIRE Download service PDOK", // TODO move to ownerRef?
				Type:  "application/opensearchdescription+xml",
			}
			*links = append(*links, link)
		}
		if template == "html" {
			href, err := replaceMustacheTemplate(ownerInfo.Spec.MetadataUrls.HTML.HrefTemplate, metadataLinks.MetadataIdentifier)
			if err != nil {
				return err
			}
			link := atomfeed.Link{
				Rel:   "related",
				Href:  href,
				Type:  "text/html",
				Title: htmlTitle,
			}
			*links = append(*links, link)
		}
	}

	return nil
}

func getDatasetLinks(atom pdoknlv3.Atom, ownerInfo smoothoperatorv1.OwnerInfo, datasetFeed pdoknlv3.DatasetFeed) ([]atomfeed.Link, error) {

	selfLink := atomfeed.Link{
		Rel:  "self",
		Href: atom.Spec.Service.BaseURL.JoinPath(datasetFeed.TechnicalName + ".xml").String(),
	}
	upLink := atomfeed.Link{
		Rel:   "up",
		Href:  atom.Spec.Service.BaseURL.JoinPath("index.xml").String(),
		Type:  "application/atom+xml",
		Title: "Top Atom Download Service Feed",
	}

	links := []atomfeed.Link{
		selfLink,
		upLink,
	}

	if datasetFeed.DatasetMetadataLinks != nil {
		err := addMetadataLinks(*datasetFeed.DatasetMetadataLinks, ownerInfo, &links, "NGR pagina voor deze dataset", false)
		if err != nil {
			return nil, err
		}
	}

	for _, link := range datasetFeed.Links {
		linkDescribedbyLink := atomfeed.Link{
			Rel:      link.Rel,
			Href:     link.Href.String(),
			Type:     link.Type,
			Hreflang: link.Hreflang,
		}
		if link.Title != nil {
			linkDescribedbyLink.Title = escapeQuotes(*link.Title)
		}
		links = append(links, linkDescribedbyLink)
	}

	return links, nil
}

func getDatasetEntries(atom pdoknlv3.Atom, datasetFeed pdoknlv3.DatasetFeed) []atomfeed.Entry {
	var entries []atomfeed.Entry
	for _, entry := range datasetFeed.Entries {

		datasetEntry := atomfeed.Entry{
			ID:       atom.Spec.Service.BaseURL.JoinPath(entry.TechnicalName + ".xml").String(),
			Link:     []atomfeed.Link{},
			Rights:   atom.Spec.Service.Rights,
			Category: []atomfeed.Category{getCategory(entry.SRS)},
			Polygon:  entry.Polygon.BBox.ToPolygon(),
		}

		if entry.Title != nil {
			datasetEntry.Title = escapeQuotes(*entry.Title)
		} else {
			datasetEntry.Title = escapeQuotes(datasetFeed.Title)
		}

		if entry.Content != nil {
			datasetEntry.Content = *entry.Content
		}

		updated := entry.Updated.In(time.FixedZone("UTC", 0)).Format(time.RFC3339)
		datasetEntry.Updated = &updated

		emptyRelCount := getEmptyRelCount(entry)
		for _, downloadLink := range entry.DownloadLinks {
			link := atomfeed.Link{
				Rel:   getDownloadLinkRel(downloadLink, emptyRelCount),
				Href:  getDownloadLinkHref(downloadLink, atom),
				Data:  getDownloadLinkData(downloadLink),
				Title: getDownloadLinkTitle(datasetFeed, entry, downloadLink),
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
		// TODO zou een "" niet eigenlijk een error moeten zijn?
		// maar dat hoort mogelijk meer bij de admission
		if downloadLink.Rel == nil || *downloadLink.Rel == "" {
			count++
		}
	}
	return
}

func getDownloadLinkRel(downloadLink pdoknlv3.DownloadLink, emptyRelCount int) (rel string) {
	switch {
	case downloadLink.Rel != nil && *downloadLink.Rel != "":
		rel = *downloadLink.Rel
	case emptyRelCount > 1:
		rel = "section"
	default:
		rel = "alternate"
	}
	return
}

func getDownloadLinkHref(downloadLink pdoknlv3.DownloadLink, atom pdoknlv3.Atom) string {
	return atom.Spec.Service.BaseURL.JoinPath("downloads", downloadLink.GetBlobName()).String()
}

// Using internal url, atom generator uses this url to determine content-length and
// content-type of the download and convert it into external url
func getDownloadLinkData(downloadLink pdoknlv3.DownloadLink) *string {
	data := pdoknlv3.GetBlobEndpoint() + "/" + downloadLink.Data
	return &data
}

func getDownloadLinkTitle(datasetFeed pdoknlv3.DatasetFeed, entry pdoknlv3.Entry, downloadLink pdoknlv3.DownloadLink) (title string) {
	if entry.Title != nil && *entry.Title != "" {
		title = *entry.Title
	} else {
		title = escapeQuotes(datasetFeed.Title)
	}
	title += "-" + downloadLink.GetBlobName()
	return
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
