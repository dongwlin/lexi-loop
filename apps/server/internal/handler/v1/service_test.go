package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dongwlin/lexi-loop/apps/server/internal/apperr"
	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
	"github.com/dongwlin/lexi-loop/apps/server/internal/handler/httpresp"
	"github.com/dongwlin/lexi-loop/apps/server/internal/service"
)

// These tests exercise the HTTP boundary without Service implementations or a database.
type wordStub struct {
	t                   *testing.T
	importWords         func(ctx context.Context, req service.ImportWordsRequest) (*service.ImportWordsResult, error)
	listWords           func(ctx context.Context, req service.ListWordsRequest) (*service.ListWordsResult, error)
	getWord             func(ctx context.Context, id uuid.UUID) (*service.WordItem, error)
	updateReviewMeaning func(ctx context.Context, id uuid.UUID, req service.UpdateReviewMeaningRequest) error
	deleteWord          func(ctx context.Context, id uuid.UUID) error
}

var _ service.Word = (*wordStub)(nil)

func (s *wordStub) ImportWords(ctx context.Context, req service.ImportWordsRequest) (*service.ImportWordsResult, error) {
	s.t.Helper()
	require.NotNil(s.t, s.importWords, "unexpected ImportWords call")
	return s.importWords(ctx, req)
}
func (s *wordStub) ListWords(ctx context.Context, req service.ListWordsRequest) (*service.ListWordsResult, error) {
	s.t.Helper()
	require.NotNil(s.t, s.listWords, "unexpected ListWords call")
	return s.listWords(ctx, req)
}
func (s *wordStub) GetWord(ctx context.Context, id uuid.UUID) (*service.WordItem, error) {
	s.t.Helper()
	require.NotNil(s.t, s.getWord, "unexpected GetWord call")
	return s.getWord(ctx, id)
}
func (s *wordStub) UpdateReviewMeaning(ctx context.Context, id uuid.UUID, req service.UpdateReviewMeaningRequest) error {
	s.t.Helper()
	require.NotNil(s.t, s.updateReviewMeaning, "unexpected UpdateReviewMeaning call")
	return s.updateReviewMeaning(ctx, id, req)
}
func (s *wordStub) DeleteWord(ctx context.Context, id uuid.UUID) error {
	s.t.Helper()
	require.NotNil(s.t, s.deleteWord, "unexpected DeleteWord call")
	return s.deleteWord(ctx, id)
}

type reviewStub struct {
	t              *testing.T
	startSession   func(ctx context.Context, req service.StartSessionRequest) (*service.StartSessionResult, error)
	submitResult   func(ctx context.Context, req service.SubmitResultRequest) error
	abandonSession func(ctx context.Context, req service.AbandonSessionRequest) error
	getSession     func(ctx context.Context, req service.GetSessionRequest) (*service.GetSessionResult, error)
}

var _ service.Review = (*reviewStub)(nil)

func (s *reviewStub) StartSession(ctx context.Context, req service.StartSessionRequest) (*service.StartSessionResult, error) {
	s.t.Helper()
	require.NotNil(s.t, s.startSession, "unexpected StartSession call")
	return s.startSession(ctx, req)
}
func (s *reviewStub) SubmitResult(ctx context.Context, req service.SubmitResultRequest) error {
	s.t.Helper()
	require.NotNil(s.t, s.submitResult, "unexpected SubmitResult call")
	return s.submitResult(ctx, req)
}
func (s *reviewStub) AbandonSession(ctx context.Context, req service.AbandonSessionRequest) error {
	s.t.Helper()
	require.NotNil(s.t, s.abandonSession, "unexpected AbandonSession call")
	return s.abandonSession(ctx, req)
}
func (s *reviewStub) GetSession(ctx context.Context, req service.GetSessionRequest) (*service.GetSessionResult, error) {
	s.t.Helper()
	require.NotNil(s.t, s.getSession, "unexpected GetSession call")
	return s.getSession(ctx, req)
}

type dictImportStub struct {
	snapshot service.DictImportSnapshot
	calls    int
}

var _ service.DictImport = (*dictImportStub)(nil)

func (s *dictImportStub) Snapshot() service.DictImportSnapshot {
	s.calls++
	return s.snapshot
}

func newMockAPI(t *testing.T, word service.Word, review service.Review, dict service.DictImport) *gin.Engine {
	t.Helper()
	httpresp.UseHumaError()
	engine := gin.New()
	config := huma.Config{
		OpenAPI: &huma.OpenAPI{OpenAPI: "3.1.0", Info: &huma.Info{Title: "test", Version: "1"}},
		Formats: huma.DefaultFormats, DefaultFormat: "application/json",
	}
	api := humagin.New(engine, config)
	NewWordHandler(word).Register(api)
	NewReviewHandler(review).Register(api)
	NewDictImportHandler(dict).Register(api)
	NewVersionHandler().Register(api)
	return engine
}

type requestMarker struct{}

func request(t *testing.T, engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), requestMarker{}, "from-http"))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func assertContext(t *testing.T, ctx context.Context) {
	t.Helper()
	assert.Equal(t, "from-http", ctx.Value(requestMarker{}))
}

func TestServiceBoundary_InvalidInputDoesNotCallService(t *testing.T) {
	id := uuid.Must(uuid.NewV7()).String()
	for _, tc := range []struct{ name, method, path, body string }{
		{"导入空列表", "POST", "/api/v1/words/import", `{"words":[]}`},
		{"导入次数非法", "POST", "/api/v1/words/import", `{"words":[{"word":"hello","count":0}]}`},
		{"详情ID非法", "GET", "/api/v1/words/not-a-uuid", ""},
		{"删除ID非法", "DELETE", "/api/v1/words/not-a-uuid", ""},
		{"更新字段类型非法", "PATCH", "/api/v1/words/" + id, `{"customReviewMeaning":123}`},
		{"开始数量非法", "POST", "/api/v1/reviews", `{"count":0}`},
		{"提交结果非法", "POST", "/api/v1/reviews/" + id + "/items/" + id, `{"result":"pending"}`},
		{"放弃ID非法", "POST", "/api/v1/reviews/not-a-uuid/abandon", ""},
		{"复习详情ID非法", "GET", "/api/v1/reviews/not-a-uuid", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine := newMockAPI(t, &wordStub{t: t}, &reviewStub{t: t}, &dictImportStub{})
			rec := request(t, engine, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), apperr.CodeValidationFailed)
		})
	}
}

func TestServiceBoundary_WordRequestsAndResponses(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	t.Run("导入参数与结果转换", func(t *testing.T) {
		calls := 0
		svc := &wordStub{t: t, importWords: func(ctx context.Context, req service.ImportWordsRequest) (*service.ImportWordsResult, error) {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, []service.ImportWordItem{{Word: "hello", Count: 2}}, req.Words)
			return &service.ImportWordsResult{Encounters: 2, Created: 1, Items: []service.ImportWordOutcome{{Word: "hello", Count: 2, Created: true}}}, nil
		}}
		rec := request(t, newMockAPI(t, svc, &reviewStub{t: t}, &dictImportStub{}), "POST", "/api/v1/words/import", `{"words":[{"word":"hello","count":2}]}`)
		require.Equal(t, 200, rec.Code)
		assert.JSONEq(t, `{"code":"OK","message":"success","data":{"encounters":2,"created":1,"updated":0,"items":[{"word":"hello","count":2,"result":"created"}]}}`, rec.Body.String())
		assert.Equal(t, 1, calls)
	})
	t.Run("列表分页与搜索", func(t *testing.T) {
		calls := 0
		svc := &wordStub{t: t, listWords: func(ctx context.Context, req service.ListWordsRequest) (*service.ListWordsResult, error) {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, service.ListWordsRequest{Page: 1, PageSize: 100, Search: "hello"}, req)
			return &service.ListWordsResult{Items: []*service.WordItem{}, Total: 0}, nil
		}}
		rec := request(t, newMockAPI(t, svc, &reviewStub{t: t}, &dictImportStub{}), "GET", "/api/v1/words?page=-1&pageSize=101&search=hello", "")
		require.Equal(t, 200, rec.Code)
		assert.Contains(t, rec.Body.String(), `"pageSize":100`)
		assert.Contains(t, rec.Body.String(), `"list":[]`)
		assert.Equal(t, 1, calls)
	})
	t.Run("详情读模型转换", func(t *testing.T) {
		calls := 0
		svc := &wordStub{t: t, getWord: func(ctx context.Context, got uuid.UUID) (*service.WordItem, error) {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, id, got)
			return &service.WordItem{UserWord: &domain.UserWord{ID: id}, Entry: &domain.DictionaryEntry{Headword: "hello"}, Phonetic: "hello", EffectiveMeaning: []domain.Meaning{{Pos: "int.", Translations: []string{"你好"}}}, MeaningSource: domain.MeaningSourceCustom}, nil
		}}
		rec := request(t, newMockAPI(t, svc, &reviewStub{t: t}, &dictImportStub{}), "GET", "/api/v1/words/"+id.String(), "")
		require.Equal(t, 200, rec.Code)
		assert.Contains(t, rec.Body.String(), `"word":"hello"`)
		assert.Contains(t, rec.Body.String(), `"meaningSource":"custom"`)
		assert.Equal(t, 1, calls)
	})
	for _, tc := range []struct {
		name, body string
		want       []domain.Meaning
	}{
		{"清除自定义", `{"customReviewMeaning":null}`, nil},
		{"保存自定义", `{"customReviewMeaning":[{"pos":"int.","translations":["你好"]}]}`, []domain.Meaning{{Pos: "int.", Translations: []string{"你好"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := &wordStub{t: t, updateReviewMeaning: func(ctx context.Context, got uuid.UUID, req service.UpdateReviewMeaningRequest) error {
				calls++
				assertContext(t, ctx)
				assert.Equal(t, id, got)
				assert.Equal(t, tc.want, req.CustomReviewMeaning)
				return nil
			}}
			rec := request(t, newMockAPI(t, svc, &reviewStub{t: t}, &dictImportStub{}), "PATCH", "/api/v1/words/"+id.String(), tc.body)
			require.Equal(t, 200, rec.Code)
			assert.JSONEq(t, `{"code":"OK","message":"success","data":{}}`, rec.Body.String())
			assert.Equal(t, 1, calls)
		})
	}
	t.Run("删除传递ID", func(t *testing.T) {
		calls := 0
		svc := &wordStub{t: t, deleteWord: func(ctx context.Context, got uuid.UUID) error {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, id, got)
			return nil
		}}
		rec := request(t, newMockAPI(t, svc, &reviewStub{t: t}, &dictImportStub{}), "DELETE", "/api/v1/words/"+id.String(), "")
		require.Equal(t, 200, rec.Code)
		assert.JSONEq(t, `{"code":"OK","message":"success","data":{}}`, rec.Body.String())
		assert.Equal(t, 1, calls)
	})
}

func TestServiceBoundary_ReviewRequestsAndResponses(t *testing.T) {
	sessionID, itemID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	t.Run("开始复习", func(t *testing.T) {
		calls := 0
		svc := &reviewStub{t: t, startSession: func(ctx context.Context, req service.StartSessionRequest) (*service.StartSessionResult, error) {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, 30, req.Count)
			return &service.StartSessionResult{SessionID: sessionID, RequestedCount: 30, TotalCount: 1, Items: []service.StartSessionItem{{ItemID: itemID, Headword: "hello"}}}, nil
		}}
		rec := request(t, newMockAPI(t, &wordStub{t: t}, svc, &dictImportStub{}), "POST", "/api/v1/reviews", `{"count":30}`)
		require.Equal(t, 200, rec.Code)
		assert.Contains(t, rec.Body.String(), `"totalCount":1`)
		assert.Contains(t, rec.Body.String(), itemID.String())
		assert.Equal(t, 1, calls)
	})
	t.Run("提交结果", func(t *testing.T) {
		calls := 0
		svc := &reviewStub{t: t, submitResult: func(ctx context.Context, req service.SubmitResultRequest) error {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, service.SubmitResultRequest{SessionID: sessionID, ItemID: itemID, Result: domain.ReviewResultRemembered}, req)
			return nil
		}}
		rec := request(t, newMockAPI(t, &wordStub{t: t}, svc, &dictImportStub{}), "POST", "/api/v1/reviews/"+sessionID.String()+"/items/"+itemID.String(), `{"result":"remembered"}`)
		require.Equal(t, 200, rec.Code)
		assert.JSONEq(t, `{"code":"OK","message":"success","data":{}}`, rec.Body.String())
		assert.Equal(t, 1, calls)
	})
	t.Run("放弃复习", func(t *testing.T) {
		calls := 0
		svc := &reviewStub{t: t, abandonSession: func(ctx context.Context, req service.AbandonSessionRequest) error {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, sessionID, req.SessionID)
			return nil
		}}
		rec := request(t, newMockAPI(t, &wordStub{t: t}, svc, &dictImportStub{}), "POST", "/api/v1/reviews/"+sessionID.String()+"/abandon", "")
		require.Equal(t, 200, rec.Code)
		assert.Equal(t, 1, calls)
	})
	t.Run("读取复习结果", func(t *testing.T) {
		calls := 0
		svc := &reviewStub{t: t, getSession: func(ctx context.Context, req service.GetSessionRequest) (*service.GetSessionResult, error) {
			calls++
			assertContext(t, ctx)
			assert.Equal(t, sessionID, req.SessionID)
			return &service.GetSessionResult{SessionID: sessionID, Status: domain.ReviewStatusCompleted, Total: 1, Remembered: 1, Items: []service.GetSessionItem{{Headword: "hello", Result: domain.ReviewResultRemembered}}}, nil
		}}
		rec := request(t, newMockAPI(t, &wordStub{t: t}, svc, &dictImportStub{}), "GET", "/api/v1/reviews/"+sessionID.String(), "")
		require.Equal(t, 200, rec.Code)
		assert.Contains(t, rec.Body.String(), `"status":"completed"`)
		assert.Contains(t, rec.Body.String(), `"remembered":1`)
		assert.Equal(t, 1, calls)
	})
}

func TestServiceBoundary_ErrorResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"资源不存在", apperr.New(apperr.NotFound, apperr.CodeNotFound, "missing", nil), 404, apperr.CodeNotFound},
		{"无可复习生词", apperr.New(apperr.FailedPrecondition, apperr.CodeNoReviewableWords, "no words", nil), 422, apperr.CodeNoReviewableWords},
		{"未知错误隐藏内部信息", errors.New("private database detail"), 500, apperr.CodeInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := &reviewStub{t: t, startSession: func(context.Context, service.StartSessionRequest) (*service.StartSessionResult, error) {
				calls++
				return nil, tc.err
			}}
			rec := request(t, newMockAPI(t, &wordStub{t: t}, svc, &dictImportStub{}), "POST", "/api/v1/reviews", `{"count":1}`)
			require.Equal(t, tc.status, rec.Code)
			var body struct {
				Code string `json:"code"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tc.code, body.Code)
			assert.NotContains(t, rec.Body.String(), "private database detail")
			assert.Equal(t, 1, calls)
		})
	}
}

func TestServiceBoundary_DictImportSnapshot(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	svc := &dictImportStub{snapshot: service.DictImportSnapshot{State: service.DictImportStateFailed, SourceVersion: "fixture", RowsProcessed: 12, RowsTotal: 20, EntriesWritten: 10, StartedAt: now, UpdatedAt: now, ErrorMessage: "invalid CSV"}}
	rec := request(t, newMockAPI(t, &wordStub{t: t}, &reviewStub{t: t}, svc), "GET", "/api/v1/dictionary-import", "")
	require.Equal(t, 200, rec.Code)
	assert.JSONEq(t, `{"code":"OK","message":"success","data":{"state":"failed","sourceVersion":"fixture","rowsProcessed":12,"rowsTotal":20,"entriesWritten":10,"startedAt":"2026-09-09T12:00:00Z","updatedAt":"2026-09-09T12:00:00Z","error":"invalid CSV"}}`, rec.Body.String())
	assert.Equal(t, 1, svc.calls)
}
