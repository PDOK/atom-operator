package v2beta1

import (
	"sync/atomic"
	"testing"

	pdoknlv3 "github.com/pdok/atom-operator/api/v3"
	smoothoperatormodel "github.com/pdok/smooth-operator/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func TestAtom_ConvertTo(t *testing.T) {
	convertFromAtom := getTestAtomV2()
	convertToAtom := &pdoknlv3.Atom{}
	dstRaw := conversion.Hub(convertToAtom)
	pdoknlv3.SetBaseURL("https://test.com/test")
	err := convertFromAtom.ConvertTo(dstRaw)
	if err != nil {
		t.Errorf("ConvertTo() error = %v", err)
	}
	testAtomV3 := getFilledAtomv3()

	if convertToAtom.APIVersion != "" {
		t.Errorf("ConvertTo() error = TypeMeta.APIVersion: %v, %v", testAtomV3.APIVersion, convertToAtom.APIVersion)
	}
	if convertToAtom.Kind != "" {
		t.Errorf("ConvertTo() error = TypeMeta.Kind: %v, %v", testAtomV3.Kind, convertToAtom.Kind)
	}

	if testAtomV3.Name != convertToAtom.Name {
		t.Errorf("ConvertTo() error = ObjectMeta.Name: %v, %v", testAtomV3.Name, convertToAtom.Name)
	}
	if len(testAtomV3.Labels) != len(convertToAtom.Labels) {
		t.Errorf("ConvertTo() error = len(testAtomV3.ObjectMeta.Labels) ")
	}
	if testAtomV3.Labels["pdok.nl/owner-id"] != convertToAtom.Labels["pdok.nl/owner-id"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label pdok.nl/owner-id: ", testAtomV3.Labels["pdok.nl/owner-id"], convertToAtom.Labels["pdok.nl/owner-id"])
	}
	if testAtomV3.Labels["pdok.nl/dataset-id"] != convertToAtom.Labels["pdok.nl/dataset-id"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label pdok.nl/dataset-id: ", testAtomV3.Labels["pdok.nl/dataset-id"], convertToAtom.Labels["pdok.nl/dataset-id"])
	}
	if testAtomV3.Labels["pdok.nl/tag"] != convertToAtom.Labels["pdok.nl/tag"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label pdok.nl/tag: ", testAtomV3.Labels["pdok.nl/tag"], convertToAtom.Labels["pdok.nl/tag"])
	}
	if testAtomV3.Labels["pdok.nl/service-type"] != convertToAtom.Labels["pdok.nl/service-type"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label pdok.nl/service-type: ", testAtomV3.Labels["pdok.nl/service-type"], convertToAtom.Labels["pdok.nl/service-type"])
	}
	if atomic.LoadInt32(testAtomV3.Spec.Lifecycle.TTLInDays) != atomic.LoadInt32(convertToAtom.Spec.Lifecycle.TTLInDays) {
		t.Errorf("ConvertTo() error = %v, expected: %d, got: %d", "TTLInDays: ", atomic.LoadInt32(testAtomV3.Spec.Lifecycle.TTLInDays), atomic.LoadInt32(convertToAtom.Spec.Lifecycle.TTLInDays))
	}

}

var testTheme = "TEST_THEME"
var TestServiceVersion = "v1_0"
var TestDataVersion = "v1.0"
var TestTTLInt = 30
var TestTTLInt32 int32 = 30
var TestUpdated = "2025-03-13T15:04:05Z"
var TestContentType = "application/pdf"
var TestLanguage = "NL"
var TestTitle = "test_title"
var TestBlobKey = "public/testme/selfservice/test.gml"

func getTestAtomV2() *Atom {

	return &Atom{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Atom",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test_me",
			Labels: map[string]string{
				"pdok.nl/owner-id":     "test_datasetowner",
				"pdok.nl/dataset-id":   "test_dataset",
				"pdok.nl/tag":          testTheme,
				"pdok.nl/service-type": "test_servicetype",
			},
		},
		Spec: AtomSpec{
			General: General{
				DatasetOwner:   "test_datasetowner",
				Dataset:        "test_dataset",
				Theme:          &testTheme,
				ServiceVersion: &TestServiceVersion,
				DataVersion:    &TestDataVersion,
			},
			Kubernetes: &Kubernetes{
				Lifecycle: &Lifecycle{
					TTLInDays: &TestTTLInt,
				},
			},
			Service: AtomService{
				Title:              "test_service_title",
				Subtitle:           "test_service_subtitle",
				MetadataIdentifier: "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy",
				Rights:             "https://creativecommons.org/publicdomain/zero/1.0/deed.nl",

				// Test the deprecated field
				//Updated: &TestUpdated,
				Author: Author{
					Name:  "somebody",
					Email: "test@gmail.com",
				},
				Datasets: []Dataset{
					{
						Name:               "test_dataset_name",
						Title:              "test_dataset_titel",
						Subtitle:           "test_dataset_subtitle",
						MetadataIdentifier: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
						SourceIdentifier:   "qqqqqqqq-qqqq-qqqq-qqqq-qqqqqqqqqqqq",
						Links: []OtherLink{
							{
								Type:        "encodingRule",
								URI:         "https://www.eionet.europa.eu/reportnet/docs/noise/guidelines/geopackage-encoding-rule-end.pdf",
								ContentType: &TestContentType,
								Language:    &TestLanguage,
							},
						},
						Downloads: []Download{
							{
								Name:    "test_download",
								Title:   &TestTitle,
								Content: &TestContentType,

								// TODO do not test both updated fields. Later switch from the one to the other.
								Updated: &TestUpdated,
								Links: []Link{
									{
										BlobKey: &TestBlobKey,
										Updated: &TestUpdated,
									},
								},
								Srs: Srs{
									URI:  "http://www.opengis.net/def/crs/EPSG/0/3035",
									Code: "ETRS89-extended / LAEA Europe",
								},
							},
						},
						Bbox: Bbox{
							Minx: 1,
							Miny: 1,
							Maxx: 2,
							Maxy: 2,
						},
					},
				},
			},
		},
	}
}

func getFilledAtomv3() *pdoknlv3.Atom {

	return &pdoknlv3.Atom{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Atom",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test_me",
			Labels: map[string]string{
				"pdok.nl/owner-id":     "test_datasetowner",
				"pdok.nl/dataset-id":   "test_dataset",
				"pdok.nl/tag":          testTheme,
				"pdok.nl/service-type": "test_servicetype",
			},
		},
		Spec: pdoknlv3.AtomSpec{
			Lifecycle: &smoothoperatormodel.Lifecycle{
				TTLInDays: &TestTTLInt32,
			},
		},
	}
}
