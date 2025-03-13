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
	err := convertFromAtom.ConvertTo(dstRaw)
	if err != nil {
		t.Errorf("ConvertTo() error = %v", err)
	}
	testAtomV3 := getFilledAtomv3()

	/*if testAtomV3.TypeMeta.APIVersion != convertToAtom.TypeMeta.APIVersion {
		// TODO: Activate next line if bug is fixed.
		// t.Errorf("ConvertTo() error = TypeMeta.APIVersion: %v, %v", testAtomV3.TypeMeta.APIVersion, convertToAtom.TypeMeta.APIVersion)
	}
	if testAtomV3.TypeMeta.Kind != convertToAtom.TypeMeta.Kind {
		// TODO: Activate next line if bug is fixed.
		// t.Errorf("ConvertTo() error = TypeMeta.Kind: %v, %v", testAtomV3.TypeMeta.Kind, convertToAtom.TypeMeta.Kind)
	}*/

	if testAtomV3.ObjectMeta.Name != convertToAtom.ObjectMeta.Name {
		t.Errorf("ConvertTo() error = ObjectMeta.Name: %v, %v", testAtomV3.ObjectMeta.Name, convertToAtom.ObjectMeta.Name)
	}
	if len(testAtomV3.ObjectMeta.Labels) != len(convertToAtom.ObjectMeta.Labels) {
		t.Errorf("ConvertTo() error = len(testAtomV3.ObjectMeta.Labels) ")
	}
	if testAtomV3.ObjectMeta.Labels["dataset-owner"] != convertToAtom.ObjectMeta.Labels["dataset-owner"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label dataset-owner: ", testAtomV3.ObjectMeta.Labels["dataset-owner"], convertToAtom.ObjectMeta.Labels["dataset-owner"])
	}
	if testAtomV3.ObjectMeta.Labels["dataset"] != convertToAtom.ObjectMeta.Labels["dataset"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label dataset: ", testAtomV3.ObjectMeta.Labels["dataset"], convertToAtom.ObjectMeta.Labels["dataset"])
	}
	if testAtomV3.ObjectMeta.Labels["theme"] != convertToAtom.ObjectMeta.Labels["theme"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label theme: ", testAtomV3.ObjectMeta.Labels["theme"], convertToAtom.ObjectMeta.Labels["theme"])
	}
	if testAtomV3.ObjectMeta.Labels["service-type"] != convertToAtom.ObjectMeta.Labels["service-type"] {
		t.Errorf("ConvertTo() error = %v, expected: %v, got: %v", "label service-type: ", testAtomV3.ObjectMeta.Labels["service-type"], convertToAtom.ObjectMeta.Labels["service-type"])
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
				"dataset-owner": "test_datasetowner",
				"dataset":       "test_dataset",
				"theme":         testTheme,
				"service-type":  "test_servicetype",
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
				"dataset-owner": "test_datasetowner",
				"dataset":       "test_dataset",
				"theme":         testTheme,
				"service-type":  "test_servicetype",
			},
		},
		Spec: pdoknlv3.AtomSpec{
			Lifecycle: smoothoperatormodel.Lifecycle{
				TTLInDays: &TestTTLInt32,
			},
		},
	}
}
