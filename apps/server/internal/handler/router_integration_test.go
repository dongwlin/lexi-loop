package handler

// Handler 组件测试：经 httptest 驱动完整 HTTP 栈（router → v1 Handler →
// Service → Repo → 真实 PostgreSQL），验证参数绑定、HTTP 状态码、统一
// 响应结构（code / message / data）与 docs/api/words.md、docs/api/reviews.md
// 的端点契约。用例共享数据库，不做并行。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envelope 是统一响应外壳；Data 用 RawMessage 保留各端点 data 的差异。
type envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// decodeEnvelope 解析统一响应外壳。
func decodeEnvelope(t *testing.T, body []byte) envelope {
	t.Helper()
	var e envelope
	require.NoError(t, json.Unmarshal(body, &e))
	return e
}

// ---- 响应 data 结构（按端点契约定义）----

type importData struct {
	Encounters int `json:"encounters"`
	Created    int `json:"created"`
	Updated    int `json:"updated"`
}

type pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
	HasMore    bool  `json:"hasMore"`
}

type listData struct {
	List []struct {
		ID   string `json:"id"`
		Word string `json:"word"`
	} `json:"list"`
	Pagination pagination `json:"pagination"`
}

type startData struct {
	SessionID      string `json:"sessionId"`
	RequestedCount int    `json:"requestedCount"`
	TotalCount     int    `json:"totalCount"`
	Items          []struct {
		ItemID string `json:"itemId"`
		Word   string `json:"word"`
	} `json:"items"`
}

type sessionData struct {
	SessionID   string  `json:"sessionId"`
	Status      string  `json:"status"`
	Total       int     `json:"total"`
	Remembered  int     `json:"remembered"`
	Forgotten   int     `json:"forgotten"`
	CompletedAt *string `json:"completedAt"`
	Items       []struct {
		Word   string `json:"word"`
		Result string `json:"result"`
	} `json:"items"`
}

func TestIntegration_WordRoutes(t *testing.T) {
	t.Run("导入生词返回统计并按契约累计", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)

		rec := doJSON(t, engine, "POST", "/api/v1/words/import", map[string]any{
			"words": []map[string]any{
				{"word": "Ambiguous", "count": 2},
				{"word": "constrain", "count": 1},
			},
		})
		require.Equal(t, http.StatusOK, rec.Code)
		e := decodeEnvelope(t, rec.Body.Bytes())
		assert.Equal(t, "OK", e.Code)
		assert.Equal(t, "success", e.Message)
		var data importData
		require.NoError(t, json.Unmarshal(e.Data, &data))
		assert.Equal(t, 3, data.Encounters)
		assert.Equal(t, 2, data.Created)
		assert.Equal(t, 0, data.Updated)

		// 大小写归一后命中同一词条 → updated；已存在词条继续累计。
		rec = doJSON(t, engine, "POST", "/api/v1/words/import", map[string]any{
			"words": []map[string]any{{"word": "ambiguous", "count": 1}},
		})
		require.Equal(t, http.StatusOK, rec.Code)
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &data))
		assert.Equal(t, 0, data.Created)
		assert.Equal(t, 1, data.Updated)
		assert.Equal(t, 1, data.Encounters)
	})

	t.Run("导入请求非法时返回 400 校验错误", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)

		// fieldErrors 是字段级错误细节的形态（HTTP API 设计规范 §3.2）。
		var fieldErrData struct {
			FieldErrors []struct {
				Field  string `json:"field"`
				Reason string `json:"reason"`
			} `json:"fieldErrors"`
		}

		t.Run("缺少 words 字段", func(t *testing.T) {
			rec := doJSON(t, engine, "POST", "/api/v1/words/import", map[string]any{})
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Header().Get("Content-Type"), "application/json",
				"错误响应 Content-Type 保持 application/json")
			e := decodeEnvelope(t, rec.Body.Bytes())
			assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", e.Code)
			require.NoError(t, json.Unmarshal(e.Data, &fieldErrData))
			assert.NotEmpty(t, fieldErrData.FieldErrors, "缺少必填字段时返回字段级错误")
		})

		t.Run("计数为零", func(t *testing.T) {
			rec := doJSON(t, engine, "POST", "/api/v1/words/import", map[string]any{
				"words": []map[string]any{{"word": "abc", "count": 0}},
			})
			require.Equal(t, http.StatusBadRequest, rec.Code)
			e := decodeEnvelope(t, rec.Body.Bytes())
			assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", e.Code)
			require.NoError(t, json.Unmarshal(e.Data, &fieldErrData))
			require.NotEmpty(t, fieldErrData.FieldErrors)
			assert.Contains(t, fieldErrData.FieldErrors[0].Field, "count")
		})

		t.Run("畸形 JSON", func(t *testing.T) {
			rec := doRaw(t, engine, "POST", "/api/v1/words/import", "{not json")
			require.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", decodeEnvelope(t, rec.Body.Bytes()).Code)
		})
	})

	t.Run("列表返回分页结构与空列表数组", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)

		rec := doJSON(t, engine, "GET", "/api/v1/words?page=1&pageSize=20", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var data listData
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &data))
		require.NotNil(t, data.List, "空列表必须是 [] 而非 null")
		assert.Empty(t, data.List)
		assert.Equal(t, 1, data.Pagination.Page)
		assert.Equal(t, 20, data.Pagination.PageSize)
		assert.Equal(t, int64(0), data.Pagination.Total)
		assert.Equal(t, 0, data.Pagination.TotalPages)
		assert.False(t, data.Pagination.HasMore)

		importWordsViaAPI(t, engine, "alpha", "beta", "gamma")

		rec = doJSON(t, engine, "GET", "/api/v1/words?page=2&pageSize=2", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &data))
		require.Len(t, data.List, 1)
		assert.Equal(t, int64(3), data.Pagination.Total)
		assert.Equal(t, 2, data.Pagination.TotalPages)
		assert.False(t, data.Pagination.HasMore, "末页没有更多")

		rec = doJSON(t, engine, "GET", "/api/v1/words?page=1&pageSize=500", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &data))
		assert.Equal(t, 100, data.Pagination.PageSize, "pageSize 超上限被钳制为 100")
	})

	t.Run("详情返回词典与学习字段及释义来源", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)
		importWordsViaAPI(t, engine, "detailword")

		var list listData
		require.NoError(t, json.Unmarshal(
			decodeEnvelope(t, doJSON(t, engine, "GET", "/api/v1/words", nil).Body.Bytes()).Data, &list))
		require.Len(t, list.List, 1)
		id := list.List[0].ID

		rec := doJSON(t, engine, "GET", "/api/v1/words/"+id, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var detail map[string]any
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &detail))
		assert.Equal(t, "detailword", detail["word"])
		assert.Equal(t, "", detail["meaningSource"], "最小词条三层皆空，来源标记为空串")
		assert.Contains(t, detail, "definition", "详情含 definition 字段（MVP 恒为空串）")
		assert.Equal(t, "", detail["definition"])

		// PATCH 自定义释义后 meaningSource 变为 custom，响应 data 为 {}。
		rec = doJSON(t, engine, "PATCH", "/api/v1/words/"+id, map[string]any{
			"customReviewMeaning": []map[string]any{
				{"pos": "noun", "translations": []string{"我的释义"}},
			},
		})
		require.Equal(t, http.StatusOK, rec.Code)
		e := decodeEnvelope(t, rec.Body.Bytes())
		assert.Equal(t, "OK", e.Code)
		assert.JSONEq(t, "{}", string(e.Data))

		rec = doJSON(t, engine, "GET", "/api/v1/words/"+id, nil)
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &detail))
		assert.Equal(t, "custom", detail["meaningSource"])

		// 传 null 清除自定义，回退为空（最小词条无任何词典释义）。
		rec = doJSON(t, engine, "PATCH", "/api/v1/words/"+id, map[string]any{"customReviewMeaning": nil})
		require.Equal(t, http.StatusOK, rec.Code)
		rec = doJSON(t, engine, "GET", "/api/v1/words/"+id, nil)
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &detail))
		assert.Equal(t, "", detail["meaningSource"])
	})

	t.Run("详情与删除按契约处理未找到与软删除", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)

		rec := doJSON(t, engine, "GET", "/api/v1/words/01991f3e-7b4c-7a10-8e3f-2c5d7a9b1001", nil)
		require.Equal(t, http.StatusNotFound, rec.Code)
		e := decodeEnvelope(t, rec.Body.Bytes())
		assert.Equal(t, "BASE.NOT_FOUND.USER", e.Code)
		assert.Equal(t, "word not found", e.Message)
		assert.JSONEq(t, "{}", string(e.Data))

		rec = doJSON(t, engine, "GET", "/api/v1/words/not-a-uuid", nil)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", decodeEnvelope(t, rec.Body.Bytes()).Code)

		importWordsViaAPI(t, engine, "deleteword")
		var list listData
		require.NoError(t, json.Unmarshal(
			decodeEnvelope(t, doJSON(t, engine, "GET", "/api/v1/words", nil).Body.Bytes()).Data, &list))
		require.Len(t, list.List, 1)
		id := list.List[0].ID

		rec = doJSON(t, engine, "DELETE", "/api/v1/words/"+id, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, "{}", string(decodeEnvelope(t, rec.Body.Bytes()).Data))

		require.Equal(t, http.StatusNotFound, doJSON(t, engine, "GET", "/api/v1/words/"+id, nil).Code,
			"软删除后详情视为不存在")

		rec = doJSON(t, engine, "DELETE", "/api/v1/words/"+id, nil)
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "word not found", decodeEnvelope(t, rec.Body.Bytes()).Message)
	})
}

func TestIntegration_ReviewRoutes(t *testing.T) {
	t.Run("无可复习生词时返回 422 业务前置错误", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)

		rec := doJSON(t, engine, "POST", "/api/v1/reviews", map[string]any{"count": 30})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		e := decodeEnvelope(t, rec.Body.Bytes())
		assert.Equal(t, "BASE.BIZ.USER_DISABLED", e.Code)
		assert.Equal(t, "no reviewable words available", e.Message)
		assert.JSONEq(t, "{}", string(e.Data))
	})

	t.Run("开始一轮返回截断后的单词列表并放弃旧轮", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)
		importWordsViaAPI(t, engine, "w1", "w2", "w3")

		rec := doJSON(t, engine, "POST", "/api/v1/reviews", map[string]any{"count": 50})
		require.Equal(t, http.StatusOK, rec.Code)
		e := decodeEnvelope(t, rec.Body.Bytes())
		assert.Equal(t, "OK", e.Code)
		var first startData
		require.NoError(t, json.Unmarshal(e.Data, &first))
		assert.Equal(t, 50, first.RequestedCount)
		assert.Equal(t, 3, first.TotalCount, "D009 截断")
		require.Len(t, first.Items, 3)
		for _, item := range first.Items {
			assert.NotEmpty(t, item.ItemID)
			assert.NotEmpty(t, item.Word)
		}

		// 开始新一轮：旧轮被放弃，经 GET 观察。
		second := startSessionViaAPI(t, engine, 2)
		assert.Equal(t, 2, second.TotalCount)
		assert.NotEqual(t, first.SessionID, second.SessionID)

		rec = doJSON(t, engine, "GET", "/api/v1/reviews/"+first.SessionID, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var got sessionData
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &got))
		assert.Equal(t, "abandoned", got.Status)
	})

	t.Run("count 缺失时返回 400", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)

		rec := doJSON(t, engine, "POST", "/api/v1/reviews", map[string]any{})
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", decodeEnvelope(t, rec.Body.Bytes()).Code)
	})

	t.Run("提交结果幂等并驱动 session 完成", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)
		importWordsViaAPI(t, engine, "s1", "s2", "s3")

		started := startSessionViaAPI(t, engine, 3)
		require.Len(t, started.Items, 3)

		// 提交第一题：200 + data {}。
		rec := doJSON(t, engine, "POST", submitPath(started.SessionID, started.Items[0].ItemID),
			map[string]any{"result": "remembered"})
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, "{}", string(decodeEnvelope(t, rec.Body.Bytes()).Data))

		// 重复提交：幂等 200。
		rec = doJSON(t, engine, "POST", submitPath(started.SessionID, started.Items[0].ItemID),
			map[string]any{"result": "remembered"})
		require.Equal(t, http.StatusOK, rec.Code)

		// 非法 result：400。
		rec = doJSON(t, engine, "POST", submitPath(started.SessionID, started.Items[1].ItemID),
			map[string]any{"result": "maybe"})
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", decodeEnvelope(t, rec.Body.Bytes()).Code)

		// 不存在的 item：契约幂等短路 200。
		rec = doJSON(t, engine, "POST",
			submitPath(started.SessionID, "01991f3e-7b4c-7a21-8e3f-2c5d7a9b2002"),
			map[string]any{"result": "forgotten"})
		require.Equal(t, http.StatusOK, rec.Code)

		// 提交剩余两题后 session 完成。
		require.Equal(t, http.StatusOK, doJSON(t, engine, "POST",
			submitPath(started.SessionID, started.Items[1].ItemID), map[string]any{"result": "forgotten"}).Code)
		require.Equal(t, http.StatusOK, doJSON(t, engine, "POST",
			submitPath(started.SessionID, started.Items[2].ItemID), map[string]any{"result": "remembered"}).Code)

		rec = doJSON(t, engine, "GET", "/api/v1/reviews/"+started.SessionID, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var got sessionData
		require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &got))
		assert.Equal(t, "completed", got.Status)
		assert.Equal(t, 3, got.Total)
		assert.Equal(t, 2, got.Remembered)
		assert.Equal(t, 1, got.Forgotten)
		require.NotNil(t, got.CompletedAt)
		require.Len(t, got.Items, 3)
		results := map[string]int{}
		for _, item := range got.Items {
			results[item.Result]++
			assert.NotEmpty(t, item.Word)
		}
		assert.Equal(t, 2, results["remembered"])
		assert.Equal(t, 1, results["forgotten"])
	})

	t.Run("获取一轮结果按契约处理未找到", func(t *testing.T) {
		resetTables(t)
		engine := newTestServer(t)

		rec := doJSON(t, engine, "GET", "/api/v1/reviews/01991f3e-7b4c-7a20-8e3f-2c5d7a9b2001", nil)
		require.Equal(t, http.StatusNotFound, rec.Code)
		e := decodeEnvelope(t, rec.Body.Bytes())
		assert.Equal(t, "BASE.NOT_FOUND.USER", e.Code)
		assert.Equal(t, "review session not found", e.Message)

		rec = doJSON(t, engine, "GET", "/api/v1/reviews/bad-id", nil)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "BASE.PARAM.VALIDATION_FAILED", decodeEnvelope(t, rec.Body.Bytes()).Code)
	})
}

// ---- 用例专属辅助 ----

// doRaw 以原始字符串为请求体执行请求（畸形 JSON 用例）。
func doRaw(t *testing.T, engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// importWordsViaAPI 经 HTTP 导入若干单词（每个 count=1）。
func importWordsViaAPI(t *testing.T, engine *gin.Engine, words ...string) {
	t.Helper()
	items := make([]map[string]any, 0, len(words))
	for _, w := range words {
		items = append(items, map[string]any{"word": w, "count": 1})
	}
	rec := doJSON(t, engine, "POST", "/api/v1/words/import", map[string]any{"words": items})
	require.Equal(t, http.StatusOK, rec.Code)
}

// startSessionViaAPI 开始一轮并解析响应。
func startSessionViaAPI(t *testing.T, engine *gin.Engine, count int) startData {
	t.Helper()
	rec := doJSON(t, engine, "POST", "/api/v1/reviews", map[string]any{"count": count})
	require.Equal(t, http.StatusOK, rec.Code)
	var data startData
	require.NoError(t, json.Unmarshal(decodeEnvelope(t, rec.Body.Bytes()).Data, &data))
	return data
}

// submitPath 组装提交结果的路径。
func submitPath(sessionID, itemID string) string {
	return "/api/v1/reviews/" + sessionID + "/items/" + itemID
}
