package mongodb

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"trpc.group/trpc-go/trpc-go/errs"
)

func TestIsDuplicateKeyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "repeated key err",
			err:  errs.NewFrameError(RetDuplicateKeyErr, "key duplicated"),
			want: true,
		},
		{
			name: "bulk write repeated key err",
			err: mongo.BulkWriteException{
				WriteConcernError: &mongo.WriteConcernError{Name: "name", Code: 100, Message: "bar"},
				WriteErrors: []mongo.BulkWriteError{
					{
						WriteError: mongo.WriteError{Code: 11000, Message: "blah E11000 blah"},
						Request:    &mongo.InsertOneModel{}},
				},
				Labels: []string{"otherError"},
			},
			want: true,
		},
		{
			name: "other err",
			err:  fmt.Errorf("E 11001, timeout"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDuplicateKeyError(tt.err); got != tt.want {
				t.Errorf("IsDuplicateKeyError() = %v, want %v", got, tt.want)
			}
		})
	}
}
