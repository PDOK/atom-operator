package generator

import (
	"reflect"
	"testing"

	"github.com/pdok/atom-generator/feeds"
	v3 "github.com/pdok/atom-operator/api/v3"
	v1 "github.com/pdok/smooth-operator/api/v1"
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
