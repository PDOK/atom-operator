package v2beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	"testing"
)

func TestAtom_ConvertTo(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       AtomSpec
		Status     AtomStatus
	}
	type args struct {
		dstRaw conversion.Hub
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		/*{
			name: "ConvertTo01",
			fields: fields{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       AtomSpec{},
				Status:     AtomStatus{},
			},
			args: args{
				dstRaw: nil,
			},
		},*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Atom{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if err := a.ConvertTo(tt.args.dstRaw); (err != nil) != tt.wantErr {
				t.Errorf("ConvertTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
