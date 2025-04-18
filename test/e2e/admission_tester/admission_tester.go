package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/pdok/atom-operator/api/v2beta1"
	v3 "github.com/pdok/atom-operator/api/v3"
	"io/fs"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
)

func main() {
	var k8sClusters string
	fmt.Print("Enter k8s-cluster folder:\n")
	fmt.Scanln(&k8sClusters)
	if !strings.HasSuffix(k8sClusters, "/") {
		k8sClusters += "/"
	}
	k8sClusters += "applications"

	v3.SetBaseURL("https://service.pdok.nl")
	err := filepath.WalkDir(k8sClusters, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, "atom.yaml") {
			checkAtom(path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("impossible to walk directories: %s", err)
	}
}
func checkAtom(path string) {
	print("Checking ")
	print(path)
	print("...")
	fileString, err := getNormalizedFileString(path)
	if err != nil {
		return
	}

	var v2Atom v2beta1.Atom
	err = yaml.Unmarshal([]byte(fileString), &v2Atom)
	if err != nil {
		fmt.Printf("Could not unmarshall '%s' to v2atom", path)
		return
	}
	var atom v3.Atom
	v2beta1.V3AtomHubFromV2(&v2Atom, &atom)

	dummy := dummyClient{}
	warnings, err := atom.ValidateCreate(dummy)
	if err != nil {
		println("ERRORS")
		println("###")
		println(err.Error())
		println("###")
	} else if len(warnings) > 0 {
		println("WARNINGS")
	} else {
		println("OK")
	}
}

type dummyClient struct {
}

func (d dummyClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return nil
}

func (d dummyClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (d dummyClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

func (d dummyClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (d dummyClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

func (d dummyClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (d dummyClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (d dummyClient) Status() client.SubResourceWriter {
	return nil
}

func (d dummyClient) SubResource(subResource string) client.SubResourceClient {
	return nil
}

func (d dummyClient) Scheme() *runtime.Scheme {
	return nil
}

func (d dummyClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (d dummyClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (d dummyClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return false, nil
}

func getNormalizedFileString(path string) (string, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Could not read file '%s', exiting", path))
	}
	fileString := string(fileBytes)
	fileString = strings.ReplaceAll(fileString, "${BLOBS_RESOURCES_BUCKET}", "resources")
	fileString = strings.ReplaceAll(fileString, "${OWNER}", "owner")
	fileString = strings.ReplaceAll(fileString, "${DATASET}", "dataset")
	fileString = strings.ReplaceAll(fileString, "${SERVICE_VERSION}", "v1_0")
	fileString = strings.ReplaceAll(fileString, "${THEME}", "theme")
	fileString = strings.ReplaceAll(fileString, "${INCLUDES}", "includes")
	fileString = strings.ReplaceAll(fileString, "${BLOBS_GEOPACKAGES_BUCKET}", "geopackages")
	fileString = strings.ReplaceAll(fileString, "${BLOBS_TIFS_BUCKET}", "tifs")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION}", "bbbb/2")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_25}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_50}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_100}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_250}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_500}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_1000}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${GPKG_VERSION_1}", "aaaa/1")
	fileString = strings.ReplaceAll(fileString, "${BLOBS_DOWNLOADS_BUCKET}", "downloads")
	fileString = strings.ReplaceAll(fileString, "${LIMITS_EPHEMERAL_STORAGE}", "102M")
	fileString = strings.ReplaceAll(fileString, "${REQUESTS_CPU}", "1001")
	fileString = strings.ReplaceAll(fileString, "${REQUESTS_MEM}", "100M")
	fileString = strings.ReplaceAll(fileString, "${REQUESTS_EPHEMERAL_STORAGE}", "101M")
	fileString = strings.ReplaceAll(fileString, "${LIMITS_MEM}", "103M")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE_10}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE_25}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE_50}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE_100}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE_250}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE_500}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${DELIVERY_DATE_1000}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${LATEST_DELIVERY_DATE}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${LATEST_DELIVERY_DATE}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_10}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_25}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_50}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_100}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_250}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_500}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_1000}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_1}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_2}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_2023}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_2022}", "2025-02-03T01:02:03Z")
	fileString = strings.ReplaceAll(fileString, "${ATOM_VERSION_2021}", "2025-02-03T01:02:03Z")

	if strings.Contains(fileString, "${") {
		reg, err := regexp.Compile("\\${.*}")
		_ = err
		println("")
		println(reg.FindString(fileString))
		return "", errors.New(fmt.Sprintf("File '%s' still has an unreplaced variable", path))
	}
	return fileString, nil
}
