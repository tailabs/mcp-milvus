package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus-proto/go-api/v2/commonpb"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/milvus-io/milvus/pkg/v2/util/merr"
	"github.com/samber/lo"
)

type CollectionInfo struct {
	BaseInfo
	Indexes  []*IndexMeta   `json:"indexes"`
	Segments []*SegmentMeta `json:"segments"`
}

type BaseInfo struct {
	CollectionId        int64    `json:"collection_id"`
	CollectionName      string   `json:"collection_name"`
	Fields              []*Field `json:"fields"`
	ShardsNum           int32    `json:"shards_num"`
	ConsistencyLevel    string   `json:"consistency_level"`
	VirtualChannelNames []string `json:"virtual_channel_names"`
	PhysicalChannels    []string `json:"physical_channel_names"`
	LoadState           string   `json:"load_state"`
	Loaded              bool     `json:"loaded"`
}

type Field struct {
	FieldID      int64  `json:"field_id"`
	Name         string `json:"name"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	DataType     string `json:"data_type"`
	ElementType  string `json:"element_type"`
	DefaultValue string `json:"default_value"`
}

type IndexMeta struct {
	Name            string            `json:"name"`
	IndexParams     map[string]string `json:"index_params"`
	UserIndexParams map[string]string `json:"user_index_params"`
	State           string            `json:"state"`
}

type SegmentMeta struct {
	SegmentID   int64  `json:"segment_id"`
	PartitionID int64  `json:"partition_id"`
	State       string `json:"state"`
	Flushed     bool   `json:"flushed"`
}

func NewMilvusGetCollectionInfoTool() mcp.Tool {
	return mcp.NewTool("milvus_get_collection_info",
		mcp.WithDescription("Lists detailed information about a specific collection"),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of collection to load."),
		),
	)
}

func MilvusGetCollectionInfoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collection, err := getCollection(ctx, cli, collectionName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	infoBytes, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format collection info: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Collection information:\n%s", string(infoBytes))), nil
}

func getCollection(ctx context.Context, cli *milvusclient.Client, collectionName string) (*CollectionInfo, error) {
	opt := milvusclient.NewDescribeCollectionOption(collectionName)
	// Loaded is always false
	// https://github.com/milvus-io/milvus/issues/34149
	collectionDesc, err := cli.DescribeCollection(ctx, opt)
	if err != nil {
		return nil, err
	}

	fields := lo.Map(collectionDesc.Schema.Fields, func(t *entity.Field, _ int) *Field {
		return &Field{
			FieldID:      t.ID,
			Name:         t.Name,
			IsPrimaryKey: t.PrimaryKey,
			DataType:     t.DataType.Name(),
			ElementType:  t.ElementType.Name(),
			DefaultValue: t.DefaultValue.String(),
		}
	})

	loadStateOpt := milvusclient.NewGetLoadStateOption(collectionName)
	loadState, err := cli.GetLoadState(ctx, loadStateOpt)
	if err != nil {
		return nil, err
	}

	indexOpt := milvusclient.NewListIndexOption(collectionName)
	indexNames, err := cli.ListIndexes(ctx, indexOpt)
	if err != nil {
		if !errors.Is(err, merr.ErrIndexNotFound) {
			return nil, err
		}
	}

	indexes := make([]*IndexMeta, 0, len(indexNames))
	for _, i := range indexNames {
		desIndexOpt := milvusclient.NewDescribeIndexOption(collectionName, i)
		indexDes, err := cli.DescribeIndex(ctx, desIndexOpt)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, &IndexMeta{
			Name: indexDes.Index.Name(),
			IndexParams: func() map[string]string {
				outer := indexDes.Index.Params()
				params, ok := outer["params"]
				if !ok {
					return outer
				}
				dst := make(map[string]string)
				if err := json.Unmarshal([]byte(params), &dst); err != nil {
					return outer
				}
				for k, v := range dst {
					outer[k] = v
				}
				delete(outer, "params")
				return outer
			}(),
			State:           commonpb.IndexState_name[int32(indexDes.State)],
			UserIndexParams: indexDes.Params(),
		})
	}

	segmentOpt := milvusclient.NewGetPersistentSegmentInfoOption(collectionName)
	segmentInfos, err := cli.GetPersistentSegmentInfo(ctx, segmentOpt)
	if err != nil {
		return nil, err
	}
	segments := lo.Map(segmentInfos, func(info *entity.Segment, _ int) *SegmentMeta {
		return &SegmentMeta{
			SegmentID:   info.ID,
			PartitionID: info.ParititionID,
			Flushed:     info.Flushed(),
			State:       info.State.String(),
		}
	})

	return &CollectionInfo{
		BaseInfo: BaseInfo{
			CollectionId:        collectionDesc.ID,
			CollectionName:      collectionDesc.Name,
			ShardsNum:           collectionDesc.ShardNum,
			Fields:              fields,
			LoadState:           commonpb.LoadState_name[int32(loadState.State)],
			Loaded:              loadState.State == entity.LoadStateLoaded,
			ConsistencyLevel:    collectionDesc.ConsistencyLevel.CommonConsistencyLevel().String(),
			VirtualChannelNames: collectionDesc.VirtualChannels,
			PhysicalChannels:    collectionDesc.PhysicalChannels,
		},
		Indexes:  indexes,
		Segments: segments,
	}, nil
}

// Tool registrar
type GetCollectionInfoTool struct{}

func (t *GetCollectionInfoTool) GetTool() mcp.Tool {
	return NewMilvusGetCollectionInfoTool()
}

func (t *GetCollectionInfoTool) GetHandler() server.ToolHandlerFunc {
	return MilvusGetCollectionInfoHandler
}

func init() {
	registry.RegisterTool(&GetCollectionInfoTool{})
}
