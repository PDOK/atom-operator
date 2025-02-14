package atom_generator

import (
	"time"
)

// AtomGeneratorConfig represents the main structure for the atom generator input
type AtomGeneratorConfig struct {
	Feeds []Feed `yaml:"feeds"`
}

// Feed represents a single feed in the configuration.
type Feed struct {
	ID         string     `yaml:"id"`
	InspireDLS string     `yaml:"inspire_dls"`
	Lang       string     `yaml:"lang,omitempty"`
	Stylesheet string     `yaml:"stylesheet"`
	Title      string     `yaml:"title"`
	Subtitle   string     `yaml:"subtitle,omitempty"`
	Link       []Link     `yaml:"link"`
	Rights     string     `yaml:"rights"`
	Updated    *time.Time `yaml:"updated,omitempty"`
	Author     Author     `yaml:"author"`
	Entry      []Entry    `yaml:"entry"`
}

// Link represents links associated with a feed or an entry.
type Link struct {
	Rel      string `yaml:"rel"`
	Href     string `yaml:"href"`
	Title    string `yaml:"title,omitempty"`
	Type     string `yaml:"type,omitempty"`
	Hreflang string `yaml:"hreflang,omitempty"`
	Version  string `yaml:"version,omitempty"`
	Time     string `yaml:"time,omitempty"`
	Bbox     string `yaml:"bbox,omitempty"`
	Data     string `yaml:"data,omitempty"`
}

// Author represents the author of the feed.
type Author struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

// Entry represents entries within a feed.
type Entry struct {
	ID                                string     `yaml:"id"`
	Title                             string     `yaml:"title"`
	SpatialDatasetIdentifierCode      string     `yaml:"spatial_dataset_identifier_code,omitempty"`
	SpatialDatasetIdentifierNamespace string     `yaml:"spatial_dataset_identifier_namespace,omitempty"`
	Link                              []Link     `yaml:"link"`
	Updated                           *time.Time `yaml:"updated,omitempty"`
	Summary                           string     `yaml:"summary,omitempty"`
	Polygon                           string     `yaml:"polygon,omitempty"`
	Category                          []Category `yaml:"category,omitempty"`
	Rights                            string     `yaml:"rights,omitempty"`
	Content                           string     `yaml:"content,omitempty"`
}

// Category represents categories for entries.
type Category struct {
	Term  string `yaml:"term"`
	Label string `yaml:"label"`
}
