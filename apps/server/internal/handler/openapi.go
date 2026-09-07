package handler

// OpenAPI 3.1 spec 的离线构建：/api/v1 的文档产物（docs/openapi/）由本文件
// 唯一生成，不挂载到线上系统（docs/specs/backend/Go 技术栈.md「API 文档」）。

import (
	"encoding/json"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"

	v1 "github.com/dongwlin/lexi-loop/apps/server/internal/handler/v1"
)

// keepBusiness422 列出契约中真实存在业务前置条件 422 的操作
// （FailedPrecondition，如 BASE.BIZ.USER_DISABLED，docs/api/reviews.md §2）。
// 其余操作的 422 响应是 huma 默认「校验失败」语义的残留——运行时已由
// httpresp.UseHumaError 降为 400，文档一并移除。
var keepBusiness422 = map[string]string{
	"start-review-session":   "业务前置条件不满足（无可复习生词，BASE.BIZ.USER_DISABLED）",
	"abandon-review-session": "业务前置条件不满足（session 已结束，BASE.BIZ.USER_DISABLED）",
}

// buildOpenAPISpec 构造 /api/v1 的 OpenAPI 3.1 spec。离线执行：不启动
// HTTP、不连接数据库；操作注册会完整执行 schema 生成与契约检查（类型 /
// 路径不合法时 panic，尽早暴露问题），而操作函数不会被调用，因此 Handler
// 的 Service 传 nil。
func buildOpenAPISpec() (*huma.OpenAPI, error) {
	api := newAPI(gin.New())
	v1.NewWordHandler(nil).Register(api)
	v1.NewReviewHandler(nil).Register(api)
	v1.NewVersionHandler().Register(api)

	spec := api.OpenAPI()
	normalizeErrorResponses(spec)
	return spec, nil
}

// normalizeErrorResponses 使 spec 与实际错误响应一致：移除 huma 自动追加
// 的 422 校验响应（运行时校验失败为 400），仅保留 keepBusiness422 中
// 声明的业务前置条件 422 并写明语义；400 等错误响应已在 Operation.Errors
// 声明并统一引用 Error 模型。另移除未声明 Errors 的操作（get-version，
// 无入参恒成功）被 huma 自动追加的 default 错误响应——对显式声明过
// 错误码的操作该删除是 no-op。
func normalizeErrorResponses(spec *huma.OpenAPI) {
	for _, item := range spec.Paths {
		for _, op := range []*huma.Operation{item.Get, item.Post, item.Put, item.Patch, item.Delete, item.Head, item.Options, item.Trace} {
			if op == nil {
				continue
			}
			delete(op.Responses, "default")
			if desc, keep := keepBusiness422[op.OperationID]; keep {
				if resp := op.Responses["422"]; resp != nil {
					resp.Description = desc
				}
				continue
			}
			delete(op.Responses, "422")
		}
	}
}

// BuildOpenAPISpec 生成 /api/v1 的 OpenAPI 3.1 spec 产物（JSON 与 YAML），
// 供 `lexi-loop openapi` 命令落盘 docs/openapi/；产物不手改，以重跑命令
// 的方式更新。
func BuildOpenAPISpec() (jsonSpec, yamlSpec []byte, err error) {
	spec, err := buildOpenAPISpec()
	if err != nil {
		return nil, nil, err
	}
	jsonSpec, err = json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	yamlSpec, err = spec.YAML()
	if err != nil {
		return nil, nil, err
	}
	return jsonSpec, yamlSpec, nil
}
