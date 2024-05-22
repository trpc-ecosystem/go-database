package mongodb

import (
	"go.mongodb.org/mongo-driver/mongo"
	"trpc.group/trpc-go/trpc-go/errs"
)

// error code, refer: https://github.com/mongodb/mongo/blob/master/src/mongo/base/error_codes.yml
const (
	RetDuplicateKeyErr = 11000 // key conflict error
)

// IsDuplicateKeyError handles whether it is a key conflict error.
func IsDuplicateKeyError(err error) bool {
	if e, ok := err.(*errs.Error); ok && e.Code == RetDuplicateKeyErr {
		return true
	}
	if e, ok := err.(mongo.BulkWriteError); ok && e.Code == RetDuplicateKeyErr {
		return true
	}
	if mongo.IsDuplicateKeyError(err) {
		return true
	}
	return false
}
