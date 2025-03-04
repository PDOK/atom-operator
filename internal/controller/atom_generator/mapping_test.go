package atom_generator

import (
	"github.com/pdok/atom-generator/feeds"
	v3 "github.com/pdok/atom-operator/api/v3"
	v1 "github.com/pdok/smooth-operator/api/v1"
	"reflect"
	"testing"
)

func TestMapAtomV3ToAtomGeneratorConfig(t *testing.T) {
	type args struct {
		atom      v3.Atom
		ownerInfo v1.OwnerInfo
	}
	tests := []struct {
		name                    string
		args                    args
		wantAtomGeneratorConfig feeds.Feeds
		wantErr                 bool
	}{
		// TODO: Add test cases.
		{
			name: "error_empty_scenario_01",
			args: args{
				atom:      v3.Atom{},
				ownerInfo: v1.OwnerInfo{},
			},
			wantAtomGeneratorConfig: feeds.Feeds{},
			wantErr:                 true,
		},
		/*		{
				name: "succesfull_scenario_02",
				args: args{
					atom: v3.Atom{
						Spec: v3.AtomSpec{
							Lifecycle: v3.Lifecycle{},
							Service: v3.Service{
								ServiceMetadataLinks: v3.MetadataLink{
									MetadataIdentifier: "7c5bbc80-d6f1-48d7-ba75-bfb0316f4f38",
									Templates:          []string{"csw", "opensearch", "html"},
								},
							},
							DatasetFeeds: []v3.DatasetFeed{
								{
									TechnicalName: "https://service.pdok.nl/bzk/geologie/bro-geotechnisch-sondeeronderzoek/atom/index.xml",
									Title:         "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
									Subtitle:      "BRO - Geotechnisch sondeeronderzoek (CPT) - Geologie (INSPIRE geharmoniseerd) ATOM",
									//Links:         []v3.Link{},
									DatasetMetadataLinks: v3.MetadataLink{
										MetadataIdentifier: "d893c05b-907e-47f2-9cbd-ceb08e68732c",
										Templates:          []string{"csw", "html"},
									},
									SpatialDatasetIdentifierCode:      "d893c05b-907e-47f2-9cbd-ceb08e68732c",
									SpatialDatasetIdentifierNamespace: "http://www.pdok.nl",
									Entries: []v3.Entry{
										{
											TechnicalName: "https://service.pdok.nl/bzk/geologie/bro-geotechnisch-sondeeronderzoek/atom/bro_geotechnisch_sondeeronderzoek_cpt_inspire_geharmoniseerd_geologie.xml",
											Title:         "BRO - Geotechnisch sondeeronderzoek (CPT) INSPIRE geharmoniseerd - Geologie",
											Content:       "Gegevens van geotechnisch sondeeronderzoek (kenset) zoals opgeslagen in de Basis Registratie Ondergrond (BRO). Geotechnisch sondeeronderzoek (in het Engels Cone Penetration Test, afgekort tot CPT) is onderzoek dat tot doel heeft informatie over de bodemkundige of geologische opbouw van de ondergrond te verwerven, waarbij in het veld metingen aan de ondergrond worden gedaan door een kegelvormige sonde de grond in te drukken. Traditioneel is het doel met de sonde de weerstand en de wrijving die de conus op de weg naar beneden ondervind te bepalen om daaruit mechanische eigenschappen van de ondergrond af te leiden. In de loop van de tijd is de sonde zo geevolueerd dat een breed scala aan metingen verricht kan worden. Voor meer informatie raadpleeg www.basisregistratieondergrond.nl",
											DownloadLinks: []v3.DownloadLink{},
										},
									},
								},
							},
						},
						Status: v3.AtomStatus{},
					},
					ownerInfo: v1.OwnerInfo{
						Spec: v1.OwnerInfoSpec{
							MetadataUrls: v1.MetadataUrls{
								CSV: v1.MetadataURL{
									HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/csw?service=CSW&version=2.0.2&request=GetRecordById&outputschema=http://www.isotc211.org/2005/gmd&elementsetname=full&id={{identifier}}",
								},
								OpenSearch: v1.MetadataURL{
									HrefTemplate: "https://www.ngr.nl/geonetwork/opensearch/dut/{{identifier}}/OpenSearchDescription.xml",
								},
								HTML: v1.MetadataURL{
									HrefTemplate: "https://www.ngr.nl/geonetwork/srv/dut/catalog.search#/metadata/{{identifier}}",
								},
							},
						},
					},
				},
				wantAtomGeneratorConfig: feeds.Feeds{},
				wantErr:                 false,
			},*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAtomGeneratorConfig, err := MapAtomV3ToAtomGeneratorConfig(tt.args.atom, tt.args.ownerInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapAtomV3ToAtomGeneratorConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotAtomGeneratorConfig, tt.wantAtomGeneratorConfig) {
				t.Errorf("MapAtomV3ToAtomGeneratorConfig() gotAtomGeneratorConfig = %v, want %v", gotAtomGeneratorConfig, tt.wantAtomGeneratorConfig)
			}
		})
	}
}
