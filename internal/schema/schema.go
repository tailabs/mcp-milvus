package schema

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/milvus-io/milvus-proto/go-api/v2/commonpb"
	"github.com/milvus-io/milvus-proto/go-api/v2/schemapb"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/samber/lo"
)

// Global lookup maps for case-insensitive DataType and FunctionType
var (
	dataTypeMap     map[string]schemapb.DataType
	functionTypeMap map[string]schemapb.FunctionType
)

// Build lookup maps supporting multiple naming conventions (e.g. camelCase, snake_case, UPPERCASE)
func init() {
	createCaseInsensitiveMap := func(sourceMap map[string]int32) map[string]int32 {
		result := lo.FlatMap(lo.Entries(sourceMap), func(entry lo.Entry[string, int32], _ int) []lo.Entry[string, int32] {
			name := entry.Key
			value := entry.Value
			// Generate all naming variants for robust lookup
			variants := []string{
				strings.ToLower(name),
				strings.ToUpper(name),
				name,
			}
			// Insert underscores before capitals for snake_case
			underscoreName := ""
			for i, r := range name {
				if i > 0 && r >= 'A' && r <= 'Z' {
					underscoreName += "_"
				}
				underscoreName += string(r)
			}
			variants = append(variants,
				strings.ToLower(underscoreName),
				strings.ToUpper(underscoreName),
				underscoreName,
			)
			uniqueVariants := lo.Uniq(variants)
			return lo.Map(uniqueVariants, func(variant string, _ int) lo.Entry[string, int32] {
				return lo.Entry[string, int32]{Key: strings.ToLower(variant), Value: value}
			})
		})
		return lo.Associate(result, func(entry lo.Entry[string, int32]) (string, int32) {
			return entry.Key, entry.Value
		})
	}
	dataTypeIntMap := createCaseInsensitiveMap(schemapb.DataType_value)
	dataTypeMap = lo.MapValues(dataTypeIntMap, func(value int32, _ string) schemapb.DataType {
		return schemapb.DataType(value)
	})
	functionTypeIntMap := createCaseInsensitiveMap(schemapb.FunctionType_value)
	functionTypeMap = lo.MapValues(functionTypeIntMap, func(value int32, _ string) schemapb.FunctionType {
		return schemapb.FunctionType(value)
	})
}

// SchemaBuilder helps build a Milvus CollectionSchema using a fluent API
// Use AddField/AddFunction and then Build()
type SchemaBuilder struct {
	schema *schemapb.CollectionSchema
}

func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		schema: &schemapb.CollectionSchema{
			Fields:    make([]*schemapb.FieldSchema, 0),
			Functions: make([]*schemapb.FunctionSchema, 0),
		},
	}
}

func (b *SchemaBuilder) WithAutoID(autoID bool) *SchemaBuilder {
	b.schema.AutoID = autoID
	return b
}

func (b *SchemaBuilder) WithDynamicField(enable bool) *SchemaBuilder {
	b.schema.EnableDynamicField = enable
	return b
}

type FieldBuilder struct {
	field  *schemapb.FieldSchema
	parent *SchemaBuilder
}

func (b *SchemaBuilder) AddField(name, description string, dataType schemapb.DataType) *FieldBuilder {
	field := &schemapb.FieldSchema{
		Name:        name,
		Description: description,
		DataType:    dataType,
		TypeParams:  make([]*commonpb.KeyValuePair, 0),
		IndexParams: make([]*commonpb.KeyValuePair, 0),
	}
	b.schema.Fields = append(b.schema.Fields, field)
	return &FieldBuilder{field: field, parent: b}
}

func (f *FieldBuilder) WithPrimaryKey(isPrimary bool) *FieldBuilder {
	f.field.IsPrimaryKey = isPrimary
	return f
}
func (f *FieldBuilder) WithAutoID(autoID bool) *FieldBuilder {
	f.field.AutoID = autoID
	return f
}
func (f *FieldBuilder) WithNullable(nullable bool) *FieldBuilder {
	f.field.Nullable = nullable
	return f
}
func (f *FieldBuilder) WithDimension(dim int) *FieldBuilder {
	f.field.TypeParams = append(f.field.TypeParams, &commonpb.KeyValuePair{
		Key:   "dim",
		Value: strconv.Itoa(dim),
	})
	return f
}
func (f *FieldBuilder) WithMaxLength(maxLen int) *FieldBuilder {
	f.field.TypeParams = append(f.field.TypeParams, &commonpb.KeyValuePair{
		Key:   "max_length",
		Value: strconv.Itoa(maxLen),
	})
	return f
}
func (f *FieldBuilder) WithTypeParam(key, value string) *FieldBuilder {
	f.field.TypeParams = append(f.field.TypeParams, &commonpb.KeyValuePair{
		Key:   key,
		Value: value,
	})
	return f
}
func (f *FieldBuilder) Done() *SchemaBuilder {
	return f.parent
}

type FunctionBuilder struct {
	function *schemapb.FunctionSchema
	parent   *SchemaBuilder
}

func (b *SchemaBuilder) AddFunction(name, description string, functionType schemapb.FunctionType) *FunctionBuilder {
	function := &schemapb.FunctionSchema{
		Name:             name,
		Description:      description,
		Type:             functionType,
		InputFieldNames:  make([]string, 0),
		OutputFieldNames: make([]string, 0),
		Params:           make([]*commonpb.KeyValuePair, 0),
	}
	b.schema.Functions = append(b.schema.Functions, function)
	return &FunctionBuilder{function: function, parent: b}
}

func (f *FunctionBuilder) WithInputFields(fieldNames ...string) *FunctionBuilder {
	f.function.InputFieldNames = append(f.function.InputFieldNames, fieldNames...)
	return f
}
func (f *FunctionBuilder) WithOutputFields(fieldNames ...string) *FunctionBuilder {
	f.function.OutputFieldNames = append(f.function.OutputFieldNames, fieldNames...)
	return f
}
func (f *FunctionBuilder) WithParam(key, value string) *FunctionBuilder {
	f.function.Params = append(f.function.Params, &commonpb.KeyValuePair{
		Key:   key,
		Value: value,
	})
	return f
}
func (f *FunctionBuilder) Done() *SchemaBuilder {
	return f.parent
}

// Build validates and returns the final schema
func (b *SchemaBuilder) Build() (*entity.Schema, error) {
	if len(b.schema.Fields) == 0 {
		return nil, fmt.Errorf("schema must contain at least one field")
	}
	if !lo.SomeBy(b.schema.Fields, func(field *schemapb.FieldSchema) bool { return field.IsPrimaryKey }) {
		return nil, fmt.Errorf("schema must have a primary key field")
	}
	return entity.NewSchema().ReadProto(b.schema), nil
}

// BuildSchemaFromMap builds a schema from a generic map (e.g. from JSON)
func BuildSchemaFromMap(schemaMap map[string]any) (*entity.Schema, error) {
	builder := NewSchemaBuilder()
	if autoID, ok := schemaMap["auto_id"].(bool); ok {
		builder.WithAutoID(autoID)
	}
	if enableDynamic, ok := schemaMap["enable_dynamic_field"].(bool); ok {
		builder.WithDynamicField(enableDynamic)
	}
	fieldsData, ok := schemaMap["fields"]
	if !ok {
		return nil, fmt.Errorf("schema must contain a 'fields' array")
	}
	fields, ok := fieldsData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'fields' must be an array")
	}
	for i, fieldData := range fields {
		fieldMap, ok := fieldData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %d must be an object", i)
		}
		// Extract required field properties
		name, ok := fieldMap["name"].(string)
		if !ok {
			return nil, fmt.Errorf("field %d missing required 'name' property", i)
		}
		dataTypeStr, ok := fieldMap["data_type"].(string)
		if !ok {
			return nil, fmt.Errorf("field %d missing required 'data_type' property", i)
		}
		dataType := stringToDataType(dataTypeStr)
		if dataType == schemapb.DataType_None {
			return nil, fmt.Errorf("field %d has unknown data type '%s'", i, dataTypeStr)
		}
		// Optional properties
		description, _ := fieldMap["description"].(string)
		fieldBuilder := builder.AddField(name, description, dataType)
		// Primary key
		if isPrimary, ok := fieldMap["is_primary"].(bool); ok && isPrimary {
			fieldBuilder.WithPrimaryKey(true)
		}
		// Auto ID
		if autoID, ok := fieldMap["auto_id"].(bool); ok {
			fieldBuilder.WithAutoID(autoID)
		}
		// Nullable
		if nullable, ok := fieldMap["nullable"].(bool); ok {
			fieldBuilder.WithNullable(nullable)
		}
		// Dimension (for vector fields)
		if dimFloat, ok := fieldMap["dimension"].(float64); ok {
			fieldBuilder.WithDimension(int(dimFloat))
		}
		// Max length (for string fields)
		if maxLenFloat, ok := fieldMap["max_length"].(float64); ok {
			fieldBuilder.WithMaxLength(int(maxLenFloat))
		}
		// Additional type parameters
		if typeParams, ok := fieldMap["type_params"].(map[string]interface{}); ok {
			for key, value := range typeParams {
				fieldBuilder.WithTypeParam(key, fmt.Sprintf("%v", value))
			}
		}
		fieldBuilder.Done()
	}
	// Handle functions if provided
	if functionsData, ok := schemaMap["functions"]; ok {
		functions, ok := functionsData.([]interface{})
		if !ok {
			return nil, fmt.Errorf("'functions' must be an array")
		}
		for i, functionData := range functions {
			functionMap, ok := functionData.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("function %d must be an object", i)
			}
			// Extract required function properties
			name, ok := functionMap["name"].(string)
			if !ok {
				return nil, fmt.Errorf("function %d missing required 'name' property", i)
			}
			functionTypeStr, ok := functionMap["type"].(string)
			if !ok {
				return nil, fmt.Errorf("function %d missing required 'type' property", i)
			}
			functionType := stringToFunctionType(functionTypeStr)
			if functionType == schemapb.FunctionType_Unknown {
				return nil, fmt.Errorf("function %d has unknown type '%s'", i, functionTypeStr)
			}
			// Optional properties
			description, _ := functionMap["description"].(string)
			functionBuilder := builder.AddFunction(name, description, functionType)
			// Input fields
			if inputFields, ok := functionMap["input_fields"].([]interface{}); ok {
				inputFieldNames := make([]string, len(inputFields))
				for j, field := range inputFields {
					if fieldName, ok := field.(string); ok {
						inputFieldNames[j] = fieldName
					} else {
						return nil, fmt.Errorf("function %d input field %d must be a string", i, j)
					}
				}
				functionBuilder.WithInputFields(inputFieldNames...)
			}
			// Output fields
			if outputFields, ok := functionMap["output_fields"].([]interface{}); ok {
				outputFieldNames := make([]string, len(outputFields))
				for j, field := range outputFields {
					if fieldName, ok := field.(string); ok {
						outputFieldNames[j] = fieldName
					} else {
						return nil, fmt.Errorf("function %d output field %d must be a string", i, j)
					}
				}
				functionBuilder.WithOutputFields(outputFieldNames...)
			}
			// Parameters
			if params, ok := functionMap["params"].(map[string]interface{}); ok {
				for key, value := range params {
					functionBuilder.WithParam(key, fmt.Sprintf("%v", value))
				}
			}
			functionBuilder.Done()
		}
	}
	return builder.Build()
}

func stringToDataType(dataType string) schemapb.DataType {
	if dt, exists := dataTypeMap[strings.ToLower(dataType)]; exists {
		return dt
	}
	return schemapb.DataType_None
}

func stringToFunctionType(functionType string) schemapb.FunctionType {
	if ft, exists := functionTypeMap[strings.ToLower(functionType)]; exists {
		return ft
	}
	return schemapb.FunctionType_Unknown
}
