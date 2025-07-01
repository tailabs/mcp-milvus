package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

func NewMilvusInsertDataTool() mcp.Tool {
	return mcp.NewTool("milvus_insert_data",
		mcp.WithDescription("Insert data into a collection."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of collection."),
		),
		mcp.WithString("data",
			mcp.Required(),
			mcp.Description("List of dictionaries, each representing a record."),
		),
	)
}

// NumericConverter defines a generic interface for numeric type conversion
type NumericConverter[T any] interface {
	Convert(float64) T
	TypeName() string
}

// Concrete implementations for different numeric types
type Int8Converter struct{}

func (Int8Converter) Convert(v float64) int8 { return int8(v) }
func (Int8Converter) TypeName() string       { return "int8" }

type Int16Converter struct{}

func (Int16Converter) Convert(v float64) int16 { return int16(v) }
func (Int16Converter) TypeName() string        { return "int16" }

type Int32Converter struct{}

func (Int32Converter) Convert(v float64) int32 { return int32(v) }
func (Int32Converter) TypeName() string        { return "int32" }

type Int64Converter struct{}

func (Int64Converter) Convert(v float64) int64 { return int64(v) }
func (Int64Converter) TypeName() string        { return "int64" }

type Float32Converter struct{}

func (Float32Converter) Convert(v float64) float32 { return float32(v) }
func (Float32Converter) TypeName() string          { return "float32" }

type Float64Converter struct{}

func (Float64Converter) Convert(v float64) float64 { return v }
func (Float64Converter) TypeName() string          { return "float64" }

// Generic function for numeric type conversion
func convertNumeric[T any](value interface{}, converter NumericConverter[T]) (T, error) {
	var zero T
	if v, ok := value.(float64); ok {
		return converter.Convert(v), nil
	}
	return zero, fmt.Errorf("expected number for %s, got %T", converter.TypeName(), value)
}

// VectorConverter defines a generic interface for vector type conversion
type VectorConverter[T any] interface {
	Convert([]float32) T
	ValidateDimension(expectedDim, actualDim int) error
	TypeName() string
}

// Concrete implementations for different vector types
type FloatVectorConverter struct{}

func (FloatVectorConverter) Convert(v []float32) []float32 { return v }
func (FloatVectorConverter) ValidateDimension(expectedDim, actualDim int) error {
	if expectedDim > 0 && actualDim != expectedDim {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d elements", expectedDim, actualDim)
	}
	return nil
}
func (FloatVectorConverter) TypeName() string { return "FloatVector" }

type BinaryVectorConverter struct{}

func (BinaryVectorConverter) Convert(v []float32) []byte {
	result := make([]byte, len(v))
	for i, f := range v {
		result[i] = byte(f)
	}
	return result
}
func (BinaryVectorConverter) ValidateDimension(expectedDim, actualDim int) error {
	expectedBytes := expectedDim / 8
	if expectedDim > 0 && actualDim != expectedBytes {
		return fmt.Errorf("binary vector dimension mismatch: expected %d bits (%d bytes), got %d bytes", expectedDim, expectedBytes, actualDim)
	}
	return nil
}
func (BinaryVectorConverter) TypeName() string { return "BinaryVector" }

type Float16VectorConverter struct{}

func (Float16VectorConverter) Convert(v []float32) entity.Float16Vector {
	return entity.FloatVector(v).ToFloat16Vector()
}
func (Float16VectorConverter) ValidateDimension(expectedDim, actualDim int) error {
	if expectedDim > 0 && actualDim != expectedDim {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d elements", expectedDim, actualDim)
	}
	return nil
}
func (Float16VectorConverter) TypeName() string { return "Float16Vector" }

type BFloat16VectorConverter struct{}

func (BFloat16VectorConverter) Convert(v []float32) entity.BFloat16Vector {
	return entity.FloatVector(v).ToBFloat16Vector()
}
func (BFloat16VectorConverter) ValidateDimension(expectedDim, actualDim int) error {
	if expectedDim > 0 && actualDim != expectedDim {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d elements", expectedDim, actualDim)
	}
	return nil
}
func (BFloat16VectorConverter) TypeName() string { return "BFloat16Vector" }

// Generic function for vector type conversion
func convertVector[T any](value interface{}, expectedDim int, converter VectorConverter[T]) (T, error) {
	var zero T
	vecSlice, ok := value.([]interface{})
	if !ok {
		return zero, fmt.Errorf("expected array for %s, got %T", converter.TypeName(), value)
	}

	// Validate dimension
	if err := converter.ValidateDimension(expectedDim, len(vecSlice)); err != nil {
		return zero, err
	}

	// Convert []interface{} to []float32 first
	floatVec := make([]float32, len(vecSlice))
	for i, elem := range vecSlice {
		if floatVal, ok := elem.(float64); ok {
			floatVec[i] = float32(floatVal)
		} else {
			return zero, fmt.Errorf("vector element at index %d: expected number, got %T", i, elem)
		}
	}

	return converter.Convert(floatVec), nil
}

// convertValueToFieldType converts JSON parsed values according to Milvus field types
func convertValueToFieldType(value interface{}, fieldType entity.FieldType, expectedDim int) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch fieldType {
	case entity.FieldTypeBool:
		if v, ok := value.(bool); ok {
			return v, nil
		}
		return nil, fmt.Errorf("expected bool, got %T", value)

	case entity.FieldTypeInt8:
		return convertNumeric(value, Int8Converter{})

	case entity.FieldTypeInt16:
		return convertNumeric(value, Int16Converter{})

	case entity.FieldTypeInt32:
		return convertNumeric(value, Int32Converter{})

	case entity.FieldTypeInt64:
		return convertNumeric(value, Int64Converter{})

	case entity.FieldTypeFloat:
		return convertNumeric(value, Float32Converter{})

	case entity.FieldTypeDouble:
		return convertNumeric(value, Float64Converter{})

	case entity.FieldTypeVarChar, entity.FieldTypeString:
		if v, ok := value.(string); ok {
			return v, nil
		}
		return nil, fmt.Errorf("expected string, got %T", value)

	case entity.FieldTypeFloatVector:
		return convertVector(value, expectedDim, FloatVectorConverter{})

	case entity.FieldTypeBinaryVector:
		return convertVector(value, expectedDim, BinaryVectorConverter{})

	case entity.FieldTypeFloat16Vector:
		return convertVector(value, expectedDim, Float16VectorConverter{})

	case entity.FieldTypeBFloat16Vector:
		return convertVector(value, expectedDim, BFloat16VectorConverter{})

	case entity.FieldTypeJSON:
		// JSON fields can accept any type, return directly
		return value, nil

	case entity.FieldTypeArray:
		// Array fields, keep as is
		return value, nil

	default:
		return nil, fmt.Errorf("unsupported field type: %v", fieldType)
	}
}

// FieldInfo holds field metadata
type FieldInfo struct {
	Type      entity.FieldType
	Dimension int
}

// SchemaInfo holds collection schema information
type SchemaInfo struct {
	Fields map[string]FieldInfo
}

// buildSchemaInfo builds schema information from collection description
func buildSchemaInfo(collectionDesc *entity.Collection) *SchemaInfo {
	schemaInfo := &SchemaInfo{
		Fields: make(map[string]FieldInfo),
	}

	for _, field := range collectionDesc.Schema.Fields {
		fieldInfo := FieldInfo{
			Type: field.DataType,
		}

		// Get dimension for vector fields
		if isVectorField(field.DataType) {
			if dimStr, exists := field.TypeParams["dim"]; exists {
				if dim, err := strconv.Atoi(dimStr); err == nil {
					fieldInfo.Dimension = dim
				}
			}
		}

		schemaInfo.Fields[field.Name] = fieldInfo
	}

	return schemaInfo
}

// isVectorField checks if a field type is a vector type
func isVectorField(fieldType entity.FieldType) bool {
	return fieldType == entity.FieldTypeFloatVector ||
		fieldType == entity.FieldTypeBinaryVector ||
		fieldType == entity.FieldTypeFloat16Vector ||
		fieldType == entity.FieldTypeBFloat16Vector
}

// transformDataForCollection transforms user data according to collection schema
func transformDataForCollection(ctx context.Context, cli *milvusclient.Client, collectionName string, data []interface{}) ([]interface{}, error) {
	// 1. Get collection schema
	opt := milvusclient.NewDescribeCollectionOption(collectionName)
	collectionDesc, err := cli.DescribeCollection(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to describe collection: %w", err)
	}

	// 2. Build schema information
	schemaInfo := buildSchemaInfo(collectionDesc)

	// 3. Transform each row of data
	transformedData := make([]interface{}, len(data))
	for i, item := range data {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("data item at index %d is not an object", i)
		}

		transformedItem := make(map[string]interface{})
		for fieldName, fieldValue := range itemMap {
			fieldInfo, exists := schemaInfo.Fields[fieldName]
			if !exists {
				// If field not in schema, might be dynamic field, keep as is
				transformedItem[fieldName] = fieldValue
				continue
			}

			convertedValue, err := convertValueToFieldType(fieldValue, fieldInfo.Type, fieldInfo.Dimension)
			if err != nil {
				return nil, fmt.Errorf("failed to convert field '%s' at row %d: %w", fieldName, i, err)
			}
			transformedItem[fieldName] = convertedValue
		}
		transformedData[i] = transformedItem
	}

	return transformedData, nil
}

func MilvusInsertDataHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dataStr, err := request.RequireString("data")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse user data
	var data []interface{}
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return mcp.NewToolResultError("Invalid data JSON: " + err.Error()), nil
	}

	// Transform data types based on schema
	transformedData, err := transformDataForCollection(ctx, cli, collectionName, data)
	if err != nil {
		return mcp.NewToolResultError("Data transformation failed: " + err.Error()), nil
	}

	// Insert data
	opt := milvusclient.NewRowBasedInsertOption(collectionName, transformedData...)
	insertResult, err := cli.Insert(ctx, opt)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Inserted Count: %d", insertResult.InsertCount)), nil
}

// Tool registrar
type InsertDataTool struct{}

func (t *InsertDataTool) GetTool() mcp.Tool {
	return NewMilvusInsertDataTool()
}

func (t *InsertDataTool) GetHandler() server.ToolHandlerFunc {
	return MilvusInsertDataHandler
}

func init() {
	registry.RegisterTool(&InsertDataTool{})
}
