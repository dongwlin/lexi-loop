package handler

// OpenAPI spec 单元测试：离线构造 spec，验证路由 / 错误模型 schema 与
// 契约一致（docs/api/words.md、docs/api/reviews.md、HTTP API 设计规范）。
// 纯单元测试，不依赖数据库，-short 模式同样运行。

import (
	"encoding/json"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// operationRefs 汇集 spec 中全部操作的 (method, path, operation)。
func operationRefs(t *testing.T, spec *huma.OpenAPI) map[string]*huma.Operation {
	t.Helper()

	ops := map[string]*huma.Operation{}
	for path, item := range spec.Paths {
		for method, op := range map[string]*huma.Operation{
			"get": item.Get, "post": item.Post, "put": item.Put,
			"patch": item.Patch, "delete": item.Delete,
		} {
			if op != nil {
				ops[method+" "+path] = op
			}
		}
	}
	return ops
}

func TestBuildOpenAPISpec(t *testing.T) {
	spec, err := buildOpenAPISpec()
	require.NoError(t, err)

	assert.Equal(t, "3.1.0", spec.OpenAPI)
	require.NotNil(t, spec.Info)
	assert.Equal(t, "LexiLoop API", spec.Info.Title)

	// ---- 路由与操作（docs/api/words.md §1、docs/api/reviews.md §1）----

	ops := operationRefs(t, spec)
	require.Len(t, ops, 9, "9 个业务操作")

	wantOps := map[string]string{
		"post /api/v1/words/import":                       "import-words",
		"get /api/v1/words":                               "list-words",
		"get /api/v1/words/{id}":                          "get-word",
		"patch /api/v1/words/{id}":                        "update-word-review-meaning",
		"delete /api/v1/words/{id}":                       "delete-word",
		"post /api/v1/reviews":                            "start-review-session",
		"post /api/v1/reviews/{sessionId}/items/{itemId}": "submit-review-result",
		"post /api/v1/reviews/{sessionId}/abandon":        "abandon-review-session",
		"get /api/v1/reviews/{id}":                        "get-review-session",
	}
	for key, opID := range wantOps {
		op, ok := ops[key]
		require.True(t, ok, "缺少操作 %s", key)
		assert.Equal(t, opID, op.OperationID)
	}

	// ---- 错误响应与 422 语义归一 ----

	for key, op := range ops {
		resp422, has422 := op.Responses["422"]
		if key == "post /api/v1/reviews" {
			assert.True(t, has422, "开始复习契约保留业务前置条件 422")
			assert.Contains(t, resp422.Description, "BASE.BIZ.USER_DISABLED")
		} else if key == "post /api/v1/reviews/{sessionId}/abandon" {
			assert.True(t, has422, "放弃复习契约保留业务前置条件 422")
			assert.Contains(t, resp422.Description, "BASE.BIZ.USER_DISABLED")
		} else {
			assert.False(t, has422, "%s 的校验失败已降为 400，不得文档化 422", key)
		}
		assert.NotContains(t, op.Responses, "default", "错误响应全部显式声明")
	}

	for _, key := range []string{
		"get /api/v1/words/{id}", "delete /api/v1/words/{id}", "get /api/v1/reviews/{id}",
		"post /api/v1/reviews/{sessionId}/abandon",
	} {
		assert.Contains(t, ops[key].Responses, "404", "%s 契约含资源不存在", key)
	}
	for key, op := range ops {
		assert.Contains(t, op.Responses, "400", "%s 契约含参数校验失败", key)
	}

	// ---- components.schemas.Error 与实际错误响应模型一致 ----

	schemas := spec.Components.Schemas.Map()
	errSchema, ok := schemas["Error"]
	require.True(t, ok, "错误模型必须以 Error 组件暴露")
	require.NotNil(t, errSchema.Properties)
	for _, field := range []string{"code", "message", "data"} {
		assert.Contains(t, errSchema.Properties, field)
	}
	dataSchema := errSchema.Properties["data"]
	dataRef := dataSchema.Ref
	if dataRef == "" && dataSchema.Items != nil {
		dataRef = dataSchema.Items.Ref
	}
	require.NotEmpty(t, dataRef, "data 引用 ErrorData 组件")
	fieldErrorsSchema := schemas["ErrorData"].Properties["fieldErrors"]
	require.NotNil(t, fieldErrorsSchema)
	assert.Equal(t, "array", fieldErrorsSchema.Type)
	fieldErrRef := fieldErrorsSchema.Items.Ref
	require.NotEmpty(t, fieldErrRef)
	fieldErrSchema := schemas["FieldError"]
	require.NotNil(t, fieldErrSchema.Properties["field"])
	require.NotNil(t, fieldErrSchema.Properties["reason"])

	// ---- 响应外壳（Envelope）与空值形态 ----

	listOp := ops["get /api/v1/words"]
	listBody := listOp.Responses["200"].Content["application/json"].Schema
	assert.NotEmpty(t, listBody.Ref, "响应 body 引用 Envelope 组件")
	assert.Contains(t, listBody.Ref, "EnvelopeListWordsResponse")

	submitOp := ops["post /api/v1/reviews/{sessionId}/items/{itemId}"]
	submitBody := submitOp.Responses["200"].Content["application/json"].Schema
	assert.Contains(t, submitBody.Ref, "EnvelopeNoData", "无数据端点 data 为空对象形态")
	assert.Equal(t, 200, submitOp.DefaultStatus, "提交结果契约返回 200 而非 204")
}

func TestBuildOpenAPISpecBytes(t *testing.T) {
	jsonSpec, yamlSpec, err := BuildOpenAPISpec()
	require.NoError(t, err)
	require.NotEmpty(t, jsonSpec)
	require.NotEmpty(t, yamlSpec)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(jsonSpec, &parsed))
	assert.Equal(t, "3.1.0", parsed["openapi"])
	assert.Contains(t, parsed, "info")
	assert.Contains(t, parsed, "components")
	assert.Contains(t, parsed, "paths")

	assert.Contains(t, string(yamlSpec), "openapi: 3.1.0")
}
