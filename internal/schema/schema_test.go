package schema

import (
	"testing"

	"github.com/milvus-io/milvus-proto/go-api/v2/schemapb"
	"github.com/stretchr/testify/assert"
)

func TestBuildSchemaFromMap(t *testing.T) {
	schemaMap := map[string]any{
		"auto_id": false,
		"fields": []any{
			map[string]any{
				"name":        "id",
				"description": "Primary key",
				"data_type":   "Int64",
				"is_primary":  true,
				"auto_id":     true,
				"nullable":    false,
			},
			map[string]any{
				"name":        "text",
				"description": "Text field",
				"data_type":   "VarChar",
				"type_params": map[string]any{
					"max_length": "1000",
				},
				"nullable": true,
			},
			map[string]any{
				"name":        "vector",
				"description": "Vector field",
				"data_type":   "FloatVector",
				"type_params": map[string]any{
					"dim": "128",
				},
				"index_params": map[string]any{
					"ivf": "flat",
				},
			},
		},
		"functions": []any{
			map[string]any{
				"name":               "bm25_function",
				"description":        "BM25 function",
				"type":               "BM25",
				"input_field_names":  []any{"text"},
				"output_field_names": []any{"vector"},
				"params": map[string]any{
					"k1": "1.2",
					"b":  "0.75",
				},
			},
		},
	}

	schema, err := BuildSchemaFromMap(schemaMap)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "id", schema.PKFieldName())
	assert.Len(t, schema.Fields, 3)

	protoSchema := schema.ProtoMessage()
	assert.Len(t, protoSchema.Functions, 1)
	assert.Equal(t, schemapb.FunctionType_BM25, protoSchema.Functions[0].Type)

	// Check nullable
	assert.False(t, protoSchema.Fields[0].Nullable)
	assert.True(t, protoSchema.Fields[1].Nullable)

	// Check type_params
	assert.Len(t, protoSchema.Fields[1].TypeParams, 1)
	assert.Equal(t, "max_length", protoSchema.Fields[1].TypeParams[0].Key)
	assert.Equal(t, "1000", protoSchema.Fields[1].TypeParams[0].Value)

	// Check index_params (should be empty, as not handled in builder)
	assert.Empty(t, protoSchema.Fields[1].IndexParams)
	// But let's check vector field's type_params
	assert.Len(t, protoSchema.Fields[2].TypeParams, 1)
	assert.Equal(t, "dim", protoSchema.Fields[2].TypeParams[0].Key)
	assert.Equal(t, "128", protoSchema.Fields[2].TypeParams[0].Value)

	// Check function params
	function := protoSchema.Functions[0]
	assert.Len(t, function.Params, 2)
	paramMap := map[string]string{}
	for _, p := range function.Params {
		paramMap[p.Key] = p.Value
	}
	assert.Equal(t, "1.2", paramMap["k1"])
	assert.Equal(t, "0.75", paramMap["b"])
}

func TestBuildSchemaFromMap_WantErr(t *testing.T) {
	t.Run("missing fields", func(t *testing.T) {
		_, err := BuildSchemaFromMap(map[string]any{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fields")
	})
	t.Run("fields not array", func(t *testing.T) {
		_, err := BuildSchemaFromMap(map[string]any{"fields": 123})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "array")
	})
	t.Run("field not object", func(t *testing.T) {
		_, err := BuildSchemaFromMap(map[string]any{"fields": []any{123}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object")
	})
	t.Run("invalid data_type", func(t *testing.T) {
		m := map[string]any{
			"fields": []any{
				map[string]any{
					"name":       "id",
					"data_type":  "INVALID",
					"is_primary": true,
				},
			},
		}
		_, err := BuildSchemaFromMap(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data type")
	})
	t.Run("no primary key", func(t *testing.T) {
		m := map[string]any{
			"fields": []any{
				map[string]any{
					"name":      "id",
					"data_type": "Int64",
				},
			},
		}
		_, err := BuildSchemaFromMap(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "primary key")
	})
	// Function type invalid
	t.Run("invalid function type", func(t *testing.T) {
		m := map[string]any{
			"fields": []any{
				map[string]any{
					"name":       "id",
					"data_type":  "Int64",
					"is_primary": true,
				},
			},
			"functions": []any{
				map[string]any{
					"name": "f1",
					"type": "INVALID",
				},
			},
		}
		_, err := BuildSchemaFromMap(m)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown type")
	})
}

func TestStringToDataType(t *testing.T) {
	tests := []struct {
		input    string
		expected schemapb.DataType
	}{
		{"Int64", schemapb.DataType_Int64},
		{"int64", schemapb.DataType_Int64},
		{"INT64", schemapb.DataType_Int64},
		{"FloatVector", schemapb.DataType_FloatVector},
		{"float_vector", schemapb.DataType_FloatVector},
		{"FLOAT_VECTOR", schemapb.DataType_FloatVector},
		{"SparseFloatVector", schemapb.DataType_SparseFloatVector},
		{"sparse_float_vector", schemapb.DataType_SparseFloatVector},
		{"INVALID", schemapb.DataType_None},
	}

	for _, test := range tests {
		result := stringToDataType(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestStringToFunctionType(t *testing.T) {
	tests := []struct {
		input    string
		expected schemapb.FunctionType
	}{
		{"BM25", schemapb.FunctionType_BM25},
		{"bm25", schemapb.FunctionType_BM25},
		{"TextEmbedding", schemapb.FunctionType_TextEmbedding},
		{"text_embedding", schemapb.FunctionType_TextEmbedding},
		{"INVALID", schemapb.FunctionType_Unknown},
	}

	for _, test := range tests {
		result := stringToFunctionType(test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestSchemaBuilder(t *testing.T) {
	schema, err := NewSchemaBuilder().
		WithAutoID(false).
		AddField("id", "Primary key", schemapb.DataType_Int64).
		WithPrimaryKey(true).
		WithAutoID(true).
		Done().
		AddField("vector", "Vector field", schemapb.DataType_FloatVector).
		WithDimension(128).
		Done().
		AddFunction("bm25", "BM25 function", schemapb.FunctionType_BM25).
		WithInputFields("text").
		WithOutputFields("vector").
		WithParam("k1", "1.2").
		Done().
		Build()

	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "id", schema.PKFieldName())
}

func TestSchemaValidation(t *testing.T) {
	t.Run("no_fields_should_fail", func(t *testing.T) {
		_, err := NewSchemaBuilder().Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one field")
	})

	t.Run("no_primary_key_should_fail", func(t *testing.T) {
		_, err := NewSchemaBuilder().
			AddField("test", "Test field", schemapb.DataType_Int64).
			Done().
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "primary key field")
	})
}

func BenchmarkBuildSchemaFromMap(b *testing.B) {
	schemaMap := map[string]any{
		"auto_id": false,
		"fields": []any{
			map[string]any{
				"name":        "id",
				"description": "Primary key",
				"data_type":   "Int64",
				"is_primary":  true,
				"auto_id":     true,
			},
			map[string]any{
				"name":        "vector",
				"description": "Vector field",
				"data_type":   "FloatVector",
				"type_params": map[string]any{
					"dim": "128",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := BuildSchemaFromMap(schemaMap)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringToDataType(b *testing.B) {
	testCases := []string{
		"Int64", "FloatVector", "VARCHAR", "float_vector",
		"BINARY_VECTOR", "sparse_float_vector", "Bool", "JSON",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, testCase := range testCases {
			stringToDataType(testCase)
		}
	}
}
