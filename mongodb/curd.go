package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoArgFilter mongo operation instruction
const (
	MongoArgFilter               = "filter"
	MongoArgSort                 = "sort"
	MongoArgSkip                 = "skip"
	MongoArgLimit                = "limit"
	MongoArgProjection           = "projection"
	MongoArgBatchSize            = "batchSize"
	MongoArgCollation            = "collation"
	MongoArgUpdate               = "update"
	MongoArgUpSert               = "upsert"
	MongoArgDoc                  = "returnDocument"
	MongoArgFieldName            = "fieldName"
	MongoArgArrayFilters         = "arrayFilters"
	MongoArgPipeline             = "pipeline"
	MongoArgMaxTimeMS            = "maxTimeMS"
	MongoArgAllowDiskUse         = "allowDiskUse"
	MongoArgDropIndexesOptions   = "DropIndexesOptions"
	MongoArgCreateIndexesOptions = "CreateIndexesOptions"
	MongoArgIndexModels          = "IndexModels"
	MongoArgIndexModel           = "IndexModel"
	MongoArgName                 = "name"
)

func executeFind(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) ([]map[string]interface{}, error) {

	filter, opts := handleArgsForExecuteFind(args)

	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	result := []map[string]interface{}{}
	for cur.Next(ctx) {
		temp := map[string]interface{}{}
		if err := bson.Unmarshal(cur.Current, &temp); err != nil {
			return result, err
		}
		result = append(result, temp)
	}
	return result, nil
}

// executeFindC returns the cursor type, and uses cursor.All/Decode to parse to the structure.
func executeFindC(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (*mongo.Cursor, error) {

	filter, opts := handleArgsForExecuteFind(args)
	return coll.Find(ctx, filter, opts)

}

// handleArgsForExecuteFind handles args for executeFind and executeFindC.
func handleArgsForExecuteFind(args map[string]interface{}) (filter map[string]interface{},
	opts *options.FindOptions) {
	opts = options.Find()
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok := opt.(map[string]interface{}); ok {
				filter = v
			}
		case MongoArgSort:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetSort(v)
			}
			if v, ok := opt.(bson.D); ok {
				opts = opts.SetSort(v)
			}
		case MongoArgSkip:
			if v, ok := opt.(float64); ok {
				opts = opts.SetSkip(int64(v))
			}
		case MongoArgLimit:
			if v, ok := opt.(float64); ok {
				opts = opts.SetLimit(int64(v))
			}
		case MongoArgProjection:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetProjection(v)
			}
		case MongoArgBatchSize:
			if v, ok := opt.(float64); ok {
				opts = opts.SetBatchSize(int32(v))
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}
	return
}

func executeDeleteOne(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (map[string]interface{}, error) {
	opts := options.Delete()
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok := opt.(map[string]interface{}); ok {
				filter = v
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}
	res, err := coll.DeleteOne(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	j, _ := json.Marshal(res)
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func executeDeleteMany(ctx context.Context, coll *mongo.Collection,
	margs map[string]interface{}) (map[string]interface{}, error) {
	mopts := options.Delete()
	var filter map[string]interface{}
	for name, opt := range margs {
		switch name {
		case MongoArgFilter:
			if v, ok := opt.(map[string]interface{}); ok {
				filter = v
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				mopts = mopts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}
	res, err := coll.DeleteMany(ctx, filter, mopts)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	j, _ := json.Marshal(res)
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func executeFindOneAndDelete(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (map[string]interface{}, error) {
	opts := options.FindOneAndDelete()
	var filter map[string]interface{}
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok := opt.(map[string]interface{}); ok {
				filter = v
			}
		case MongoArgSort:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetSort(v)
			}
			if v, ok := opt.(bson.D); ok {
				opts = opts.SetSort(v)
			}
		case MongoArgProjection:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetProjection(v)
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}
	cur := coll.FindOneAndDelete(ctx, filter, opts)

	result := map[string]interface{}{}

	if err := cur.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func executeFindOneAndUpdate(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (map[string]interface{}, error) {

	cur, _ := executeFindOneAndUpdateS(ctx, coll, args)

	result := map[string]interface{}{}

	if err := cur.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// executeFindOneAndUpdateC returns mongo.SingleResult type, and uses Decode to parse to structure.
func executeFindOneAndUpdateS(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (*mongo.SingleResult, error) {
	filter, update, fupdatePipe, fopts := handleArgsForOneAndUpdateS(args)
	var cur *mongo.SingleResult
	if fupdatePipe != nil {
		cur = coll.FindOneAndUpdate(ctx, filter, fupdatePipe, fopts)
	} else {
		cur = coll.FindOneAndUpdate(ctx, filter, update, fopts)
	}

	return cur, nil
}

// handleArgsForOneAndUpdateS is an auxiliary function that handles the args passed in executeFindOneAndUpdateS.
func handleArgsForOneAndUpdateS(args map[string]interface{}) (filter map[string]interface{},
	update map[string]interface{}, fupdatePipe []interface{}, fopts *options.FindOneAndUpdateOptions) {
	fopts = options.FindOneAndUpdate()
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok := opt.(map[string]interface{}); ok {
				filter = v
			}
		case MongoArgUpdate:
			var ok bool
			update, ok = opt.(map[string]interface{})
			if !ok {
				if v, ok := opt.([]interface{}); ok {
					fupdatePipe = v
				}
			}
		case MongoArgArrayFilters:
			if v, ok := opt.([]interface{}); ok {
				fopts = fopts.SetArrayFilters(options.ArrayFilters{
					Filters: v,
				})
			}
		case MongoArgSort:
			if v, ok := opt.(map[string]interface{}); ok {
				fopts = fopts.SetSort(v)
			}
			if v, ok := opt.(bson.D); ok {
				fopts = fopts.SetSort(v)
			}
		case MongoArgProjection:
			if v, ok := opt.(map[string]interface{}); ok {
				fopts = fopts.SetProjection(v)
			}
		case MongoArgUpSert:
			if v, ok := opt.(bool); ok {
				fopts = fopts.SetUpsert(v)
			}
		case MongoArgDoc:
			if v, ok := opt.(string); ok {
				fopts = setReturnDocument(fopts, v)
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				fopts = fopts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}
	return
}

func setReturnDocument(fopts *options.FindOneAndUpdateOptions, rdType string) *options.FindOneAndUpdateOptions {
	switch rdType {
	case "After":
		fopts = fopts.SetReturnDocument(options.After)
	case "Before":
		fopts = fopts.SetReturnDocument(options.Before)
	default:
		// do nothing
	}
	return fopts
}

func executeInsertOne(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (map[string]interface{}, error) {
	if _, ok := args["document"]; !ok {
		return nil, fmt.Errorf("InsertOne args error,need key document")
	}

	document, ok := args["document"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("InsertOne document type error,need map")
	}

	res, err := coll.InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	j, _ := json.Marshal(res)
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func executeInsertMany(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (map[string]interface{}, error) {

	if _, ok := args["documents"]; !ok {
		return nil, fmt.Errorf("InsertMany args error,need key document")
	}
	documents, ok := args["documents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("InsertMany document type error,need slice")
	}
	for i, doc := range documents {
		docM, ok := doc.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("InsertMany document element type error,need map[string]interface{}")
		}

		documents[i] = docM
	}
	res, err := coll.InsertMany(ctx, documents)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	j, _ := json.Marshal(res)
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func executeUpdateOne(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (map[string]interface{}, error) {
	uopts := options.Update()
	var ufilter map[string]interface{}
	var update map[string]interface{}
	var updatePipe []interface{}
	var ok bool
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok1 := opt.(map[string]interface{}); ok1 {
				ufilter = v
			}
		case MongoArgUpdate:
			update, ok = opt.(map[string]interface{})
			if !ok {
				if v, ok := opt.([]interface{}); ok {
					updatePipe = v
				}
			}
		case MongoArgArrayFilters:
			if v, ok := opt.([]interface{}); ok {
				uopts = uopts.SetArrayFilters(options.ArrayFilters{
					Filters: v,
				})
			}
		case MongoArgUpSert:
			if v, ok := opt.(bool); ok {
				uopts = uopts.SetUpsert(v)
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				uopts = uopts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}

	var upRes *mongo.UpdateResult
	var err error
	if updatePipe != nil {
		upRes, err = coll.UpdateOne(ctx, ufilter, updatePipe, uopts)
	} else {
		upRes, err = coll.UpdateOne(ctx, ufilter, update, uopts)
	}
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	j, _ := json.Marshal(upRes)
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func executeUpdateMany(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) (map[string]interface{}, error) {
	opts := options.Update()
	var filter map[string]interface{}
	var update map[string]interface{}
	var updatePipe []interface{}
	var ok bool
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok1 := opt.(map[string]interface{}); ok1 {
				filter = v
			}
		case MongoArgUpdate:
			update, ok = opt.(map[string]interface{})
			if !ok {
				if v, ok := opt.([]interface{}); ok {
					updatePipe = v
				}
			}
		case MongoArgArrayFilters:
			if v, ok := opt.([]interface{}); ok {
				opts = opts.SetArrayFilters(options.ArrayFilters{
					Filters: v,
				})
			}
		case MongoArgUpSert:
			if v, ok := opt.(bool); ok {
				opts = opts.SetUpsert(v)
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}

	var upRes *mongo.UpdateResult
	var err error
	if updatePipe != nil {
		upRes, err = coll.UpdateMany(ctx, filter, updatePipe, opts)
	} else {
		upRes, err = coll.UpdateMany(ctx, filter, update, opts)
	}
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	j, _ := json.Marshal(upRes)
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func executeCount(ctx context.Context, coll *mongo.Collection, args map[string]interface{}) (int64, error) {
	var filter map[string]interface{}
	opts := options.Count()
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok := opt.(map[string]interface{}); ok {
				filter = v
			}
		case MongoArgSkip:
			if v, ok := opt.(float64); ok {
				opts = opts.SetSkip(int64(v))
			}
		case MongoArgLimit:
			if v, ok := opt.(float64); ok {
				opts = opts.SetLimit(int64(v))
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}
	return coll.CountDocuments(ctx, filter, opts)
}

func executeDistinct(ctx context.Context, coll *mongo.Collection,
	args map[string]interface{}) ([]interface{}, error) {
	var fieldName string
	var filter map[string]interface{}
	opts := options.Distinct()
	for name, opt := range args {
		switch name {
		case MongoArgFilter:
			if v, ok := opt.(map[string]interface{}); ok {
				filter = v
			}
		case MongoArgFieldName:
			if v, ok := opt.(string); ok {
				fieldName = v
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetCollation(collationFromMap(v))
			}
		default:
		}
	}
	return coll.Distinct(ctx, fieldName, filter, opts)
}

func executeAggregate(ctx context.Context, coll *mongo.Collection, args map[string]interface{}) (
	[]map[string]interface{}, error) {

	cur, err := executeAggregateC(ctx, coll, args)
	if err != nil {
		return nil, err
	}

	result := []map[string]interface{}{}
	for cur.Next(ctx) {
		temp := map[string]interface{}{}
		if err := bson.Unmarshal(cur.Current, &temp); err != nil {
			return result, err
		}
		result = append(result, temp)
	}
	return result, nil
}

func executeAggregateC(ctx context.Context, coll *mongo.Collection, args map[string]interface{}) (
	*mongo.Cursor, error) {

	var pipeline []interface{}
	opts := options.Aggregate()
	for name, opt := range args {
		switch name {

		case MongoArgPipeline:
			p, ok := opt.([]interface{})
			if !ok {
				return nil, fmt.Errorf("Aggregate args error,args value need slice")
			}
			pipeline = p
		case MongoArgBatchSize:
			if v, ok := opt.(float64); ok {
				opts = opts.SetBatchSize(int32(v))
			}
		case MongoArgCollation:
			if v, ok := opt.(map[string]interface{}); ok {
				opts = opts.SetCollation(collationFromMap(v))
			}
		case MongoArgMaxTimeMS:
			if v, ok := opt.(float64); ok {
				opts = opts.SetMaxTime(time.Duration(v) * time.Millisecond)
			}
		default:
		}
	}
	return coll.Aggregate(ctx, pipeline, opts)
}
func collationFromMap(m map[string]interface{}) *options.Collation {
	var collation options.Collation

	if locale, found := m["locale"]; found {
		if v, ok := locale.(string); ok {
			collation.Locale = v
		}
	}

	if caseLevel, found := m["caseLevel"]; found {
		if v, ok := caseLevel.(bool); ok {
			collation.CaseLevel = v
		}
	}

	if caseFirst, found := m["caseFirst"]; found {
		if v, ok := caseFirst.(string); ok {
			collation.CaseFirst = v
		}
	}

	if strength, found := m["strength"]; found {
		if v, ok := strength.(float64); ok {
			collation.Strength = int(v)
		}
	}

	if numericOrdering, found := m["numericOrdering"]; found {
		if v, ok := numericOrdering.(bool); ok {
			collation.NumericOrdering = v
		}
	}

	if alternate, found := m["alternate"]; found {
		if v, ok := alternate.(string); ok {
			collation.Alternate = v
		}
	}

	if maxVariable, found := m["maxVariable"]; found {
		if v, ok := maxVariable.(string); ok {
			collation.MaxVariable = v
		}
	}

	if normalization, found := m["normalization"]; found {
		if v, ok := normalization.(bool); ok {
			collation.Normalization = v
		}
	}

	if backwards, found := m["backwards"]; found {
		if v, ok := backwards.(bool); ok {
			collation.Backwards = v
		}
	}

	return &collation
}

func executeBulkWrite(ctx context.Context, coll *mongo.Collection, args map[string]interface{},
) (map[string]interface{}, error) {
	if _, ok := args["documents"]; !ok {
		return nil, fmt.Errorf("InsertMany args error,need key document")
	}
	operations, ok := args["documents"].([]mongo.WriteModel)
	if !ok {
		return nil, fmt.Errorf("InsertMany document type error,need slice")
	}
	bulkOption := &options.BulkWriteOptions{}
	bulkOption.SetOrdered(false)
	if optRaw, ok := args["BulkWriteOptions"]; ok {
		if opt, ok := optRaw.(*options.BulkWriteOptions); ok {
			bulkOption = opt
		}
	}
	res, err := coll.BulkWrite(ctx, operations, bulkOption)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	j, _ := json.Marshal(res)
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func executeIndexCreateOne(ctx context.Context, iv mongo.IndexView, args map[string]interface{}) (string, error) {
	indexModel, ok := args[MongoArgIndexModel]
	if !ok {
		return "", errors.New("CreateOne args error,need key IndexModel")
	}
	model, ok := indexModel.(mongo.IndexModel)
	if !ok {
		return "", errors.New("IndexModel document type error,need mongo.IndexModel")
	}
	createIndexesOption := &options.CreateIndexesOptions{}
	if optRaw, ok := args[MongoArgCreateIndexesOptions]; ok {
		if opt, ok := optRaw.(*options.CreateIndexesOptions); ok {
			createIndexesOption = opt
		}
	}
	return iv.CreateOne(ctx, model, createIndexesOption)
}

func executeIndexCreateMany(ctx context.Context, iv mongo.IndexView, args map[string]interface{}) ([]string, error) {
	indexModels, ok := args[MongoArgIndexModels]
	if !ok {
		return nil, errors.New("CreateMany args error,need key IndexModels")
	}
	models, ok := indexModels.([]mongo.IndexModel)
	if !ok {
		return nil, errors.New("IndexModel document type error,need []mongo.IndexModel")
	}
	createIndexesOption := &options.CreateIndexesOptions{}
	if optRaw, ok := args[MongoArgCreateIndexesOptions]; ok {
		if opt, ok := optRaw.(*options.CreateIndexesOptions); ok {
			createIndexesOption = opt
		}
	}
	return iv.CreateMany(ctx, models, createIndexesOption)
}

func executeIndexDropOne(ctx context.Context, iv mongo.IndexView, args map[string]interface{}) (bson.Raw, error) {
	nameObj, ok := args[MongoArgName]
	if !ok {
		return nil, errors.New("DropOn args error,need key name")
	}
	name, ok := nameObj.(string)
	if !ok {
		return nil, errors.New("IndexModel document type error,need string")
	}
	dropIndexesOption := &options.DropIndexesOptions{}
	if optRaw, ok := args[MongoArgDropIndexesOptions]; ok {
		if opt, ok := optRaw.(*options.DropIndexesOptions); ok {
			dropIndexesOption = opt
		}
	}
	return iv.DropOne(ctx, name, dropIndexesOption)
}

func executeIndexDropAll(ctx context.Context, iv mongo.IndexView, args map[string]interface{}) (bson.Raw, error) {
	dropIndexesOption := &options.DropIndexesOptions{}
	if optRaw, ok := args[MongoArgDropIndexesOptions]; ok {
		if opt, ok := optRaw.(*options.DropIndexesOptions); ok {
			dropIndexesOption = opt
		}
	}
	return iv.DropAll(ctx, dropIndexesOption)
}
