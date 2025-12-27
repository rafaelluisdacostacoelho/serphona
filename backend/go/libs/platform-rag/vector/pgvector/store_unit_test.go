package pgvector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/model"
)

type failRegisterer struct{ err error }

func (f failRegisterer) Register(prometheus.Collector) error { return f.err }

func (f failRegisterer) MustRegister(...prometheus.Collector) {}

func (f failRegisterer) Unregister(prometheus.Collector) bool { return true }

func TestTableNameAndDimensionDefaults(t *testing.T) {
	s := Store{config: Config{}}
	name, err := s.tableName()
	if err != nil || name != "rag_chunks" {
		t.Fatalf("expected default table name, got %s err=%v", name, err)
	}
	if dim := s.dimension(); dim != 1536 {
		t.Fatalf("expected default dimension 1536, got %d", dim)
	}
}

func TestTableNameInvalid(t *testing.T) {
	s := Store{config: Config{TableName: "bad-name!"}}
	if _, err := s.tableName(); err == nil {
		t.Fatalf("expected table name validation error")
	}
}

func TestCreateSchemaExecutesStatements(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE EXTENSION").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE INDEX").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE INDEX").WillReturnResult(sqlmock.NewResult(0, 0))

	cfg := Config{TableName: "rag_chunks", Dimension: 4, Lists: 10}
	if err := CreateSchema(context.Background(), db, cfg); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertChunksSuccess(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	cfg := Config{TableName: "rag_chunks", Dimension: 3}
	s := NewStore(db, cfg)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO rag_chunks").WithArgs(
		"t1", "ns1", "doc1", "c1", "hello", sqlmock.AnyArg(), sqlmock.AnyArg(), "etag", sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	chunks := []model.Chunk{{
		TenantID:   "t1",
		Namespace:  "ns1",
		DocumentID: "doc1",
		ChunkID:    "c1",
		Content:    "hello",
		Metadata:   model.ChunkMetadata{Tags: []string{"faq"}},
		Embedding:  []float32{0.1, 0.2, 0.3},
		ETag:       "etag",
	}}

	if err := s.UpsertChunks(context.Background(), chunks); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertChunksUsesProvidedCreatedAt(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	s := NewStore(db, Config{TableName: "rag_chunks", Dimension: 1})
	created := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO rag_chunks").WithArgs(
		"t", "n", "", "c", "body", sqlmock.AnyArg(), sqlmock.AnyArg(), "", created,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	chunk := model.Chunk{TenantID: "t", Namespace: "n", ChunkID: "c", Content: "body", Embedding: []float32{1}, CreatedAt: created}
	if err := s.UpsertChunks(context.Background(), []model.Chunk{chunk}); err != nil {
		t.Fatalf("upsert with created_at: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertChunksBadDimension(t *testing.T) {
	s := NewStore(nil, Config{Dimension: -1})
	chunks := []model.Chunk{{TenantID: "t", Namespace: "n", ChunkID: "c", Embedding: []float32{1}}}
	if err := s.UpsertChunks(context.Background(), chunks); err == nil {
		t.Fatalf("expected dimension error")
	}
}

func TestQuerySuccessWithFiltersAndScore(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	cfg := Config{TableName: "rag_chunks", Dimension: 3}
	s := NewStore(db, cfg)

	rows := sqlmock.NewRows([]string{"tenant_id", "namespace", "document_id", "chunk_id", "content", "metadata", "etag", "created_at", "distance"}).
		AddRow("t1", "ns1", "doc1", "c1", "hello", []byte(`{"tags":["faq"],"lang":"en"}`), "etag", time.Now(), 0.2)

	mock.ExpectQuery("SELECT tenant_id").WillReturnRows(rows)

	q := model.Query{
		TenantID:    "t1",
		Namespace:   "ns1",
		QueryVector: []float32{0.1, 0.2, 0.3},
		TopK:        1,
		Filters:     model.Filters{Attributes: map[string]string{"lang": "en"}},
		MinScore:    0,
	}
	res, err := s.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 1 || res[0].ChunkID != "c1" || res[0].Score <= 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res[0].Metadata.Attributes["lang"] != "en" {
		t.Fatalf("metadata not unmarshaled: %+v", res[0].Metadata)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestQueryWithoutFilters(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	s := NewStore(db, Config{TableName: "rag_chunks", Dimension: 1})
	rows := sqlmock.NewRows([]string{"tenant_id", "namespace", "document_id", "chunk_id", "content", "metadata", "etag", "created_at", "distance"}).
		AddRow("t", "n", "d", "c", "body", []byte(`{"lang":"en"}`), "", time.Now(), 0.1)
	mock.ExpectQuery("SELECT tenant_id").WillReturnRows(rows)

	q := model.Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1}}
	res, err := s.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 1 || res[0].Metadata.Attributes["lang"] != "en" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestQueryDimensionMismatch(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	s := NewStore(db, Config{Dimension: 2})
	q := model.Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1}}
	if _, err := s.Query(context.Background(), q); err == nil {
		t.Fatalf("expected dimension mismatch error")
	}
	_ = mock.ExpectationsWereMet()
}

func TestPingWrapsError(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
	defer db.Close()

	s := NewStore(db, Config{})
	mock.ExpectPing().WillReturnError(errors.New("boom"))

	if err := s.Ping(context.Background()); err == nil {
		t.Fatalf("expected ping error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestObserverCoverage(t *testing.T) {
	ctx, end := NoopObserver{}.Trace(context.Background(), "op")
	end(nil)
	NoopObserver{}.RecordLatency(ctx, "op", time.Millisecond, nil)
}

func TestMarshalFiltersNilAndData(t *testing.T) {
	buf, err := marshalFilters(model.Filters{})
	if err != nil || buf != nil {
		t.Fatalf("expected nil filters result")
	}

	buf, err = marshalFilters(model.Filters{Tags: []string{"a", "a"}, Attributes: map[string]string{"lang": "en"}})
	if err != nil || buf == nil {
		t.Fatalf("expected marshaled filters, err=%v", err)
	}
}

func TestMarshalFiltersAllFields(t *testing.T) {
	buf, err := marshalFilters(model.Filters{
		Tags:       []string{"a"},
		ACL:        []string{"admin"},
		Attributes: map[string]string{"lang": "en"},
		Source:     "kb",
		Version:    "v1",
		URI:        "https://example.com",
		Language:   "pt",
		Channel:    "email",
	})
	if err != nil {
		t.Fatalf("marshal filters: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"tags", "acl", "lang", "source", "version", "uri", "language", "channel"} {
		if _, ok := got[k]; !ok {
			t.Fatalf("expected key %s in filters", k)
		}
	}
}

func TestWithTimeoutZero(t *testing.T) {
	ctx := context.Background()
	got, cancel := withTimeout(ctx, 0)
	cancel()
	if got != ctx {
		t.Fatalf("expected original context when timeout zero")
	}
}

// Ensure pgvector.NewVector implements driver.Valuer to satisfy sqlmock args during tests.
func TestPgvectorValuer(t *testing.T) {
	v := pgvector.NewVector([]float32{1})
	if _, err := v.Value(); err != nil {
		t.Fatalf("valuer error: %v", err)
	}
}

func TestCreateSchemaDefaultsAndError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	// Defaults apply when cfg fields are zero.
	mock.ExpectExec("CREATE EXTENSION").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE INDEX").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE INDEX").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := CreateSchema(context.Background(), db, Config{}); err != nil {
		t.Fatalf("default create schema: %v", err)
	}

	// Fail fast on exec error.
	mock.ExpectExec("CREATE EXTENSION").WillReturnError(errors.New("fail"))
	if err := CreateSchema(context.Background(), db, Config{TableName: "ok"}); err == nil {
		t.Fatalf("expected exec error")
	}
	_ = mock.ExpectationsWereMet()
}

func TestCreateSchemaInvalidTableName(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	if err := CreateSchema(context.Background(), db, Config{TableName: "bad-name!"}); err == nil {
		t.Fatalf("expected invalid table name error")
	}
}

func TestUpsertChunksEdgeErrors(t *testing.T) {
	// Empty input short-circuits.
	if err := NewStore(nil, Config{Dimension: 1}).UpsertChunks(context.Background(), nil); err != nil {
		t.Fatalf("empty chunks should succeed: %v", err)
	}

	// Invalid table name fails fast.
	if err := NewStore(nil, Config{TableName: "bad-name!", Dimension: 1}).UpsertChunks(context.Background(), []model.Chunk{{TenantID: "t", Namespace: "n", ChunkID: "c", Embedding: []float32{1}}}); err == nil {
		t.Fatalf("expected invalid table name error")
	}

	// Validation error before DB work.
	badChunk := model.Chunk{TenantID: "t", Namespace: "", ChunkID: "c", Embedding: []float32{1}}
	if err := NewStore(nil, Config{Dimension: 1}).UpsertChunks(context.Background(), []model.Chunk{badChunk}); err == nil {
		t.Fatalf("expected validation error before DB")
	}

	// BeginTx failure.
	db, mock, _ := sqlmock.New()
	defer db.Close()
	s := NewStore(db, Config{TableName: "rag_chunks", Dimension: 1})
	mock.ExpectBegin().WillReturnError(errors.New("begin fail"))
	chunk := model.Chunk{TenantID: "t", Namespace: "n", ChunkID: "c", Embedding: []float32{1}, Content: "c"}
	if err := s.UpsertChunks(context.Background(), []model.Chunk{chunk}); err == nil {
		t.Fatalf("expected begin error")
	}

	// Exec error.
	db2, mock2, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	defer db2.Close()
	s = NewStore(db2, Config{TableName: "rag_chunks", Dimension: 1})
	mock2.ExpectBegin()
	mock2.ExpectExec("INSERT INTO rag_chunks").WillReturnError(errors.New("exec fail"))
	mock2.ExpectRollback()
	if err := s.UpsertChunks(context.Background(), []model.Chunk{chunk}); err == nil {
		t.Fatalf("expected exec error")
	}

	// Commit error wrapped in StoreError.
	db3, mock3, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	defer db3.Close()
	s = NewStore(db3, Config{TableName: "rag_chunks", Dimension: 1})
	mock3.ExpectBegin()
	mock3.ExpectExec("INSERT INTO rag_chunks").WillReturnResult(sqlmock.NewResult(1, 1))
	mock3.ExpectCommit().WillReturnError(errors.New("commit fail"))
	if err := s.UpsertChunks(context.Background(), []model.Chunk{chunk}); err == nil {
		t.Fatalf("expected commit failure wrapped")
	} else {
		var se StoreError
		if !errors.As(err, &se) {
			t.Fatalf("expected StoreError wrapper, got %T", err)
		}
	}
}

func TestQueryScoreFilteringAndErrors(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	defer db.Close()
	s := NewStore(db, Config{Dimension: 2})

	rows := sqlmock.NewRows([]string{"tenant_id", "namespace", "document_id", "chunk_id", "content", "metadata", "etag", "created_at", "distance"}).
		AddRow("t", "n", "d", "c", "content", []byte(`{"tags":["x"]}`), "etag", time.Now(), 99)
	mock.ExpectQuery("SELECT tenant_id").WillReturnRows(rows)

	q := model.Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1, 2}, MinScore: 0.9}
	got, err := s.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected filter to drop rows")
	}

	// Scan error path.
	mock.ExpectQuery("SELECT tenant_id").WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(1))
	if _, err := s.Query(context.Background(), q); err == nil {
		t.Fatalf("expected scan error")
	}

	// rows.Err path.
	badRows := sqlmock.NewRows([]string{"tenant_id", "namespace", "document_id", "chunk_id", "content", "metadata", "etag", "created_at", "distance"}).
		AddRow("t", "n", "d", "c", "content", []byte(`{"tags":["x"]}`), "etag", time.Now(), 0)
	badRows.RowError(0, errors.New("row err"))
	mock.ExpectQuery("SELECT tenant_id").WillReturnRows(badRows)
	if _, err := s.Query(context.Background(), q); err == nil {
		t.Fatalf("expected rows err")
	}
	_ = mock.ExpectationsWereMet()
}

func TestQueryDatabaseErrorAndDefaults(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	defer db.Close()
	s := NewStore(db, Config{Dimension: 1})
	mock.ExpectQuery("SELECT tenant_id").WillReturnError(errors.New("db err"))
	q := model.Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1}}
	if _, err := s.Query(context.Background(), q); err == nil {
		t.Fatalf("expected store error on query failure")
	} else {
		var se StoreError
		if !errors.As(err, &se) || se.Op != "query" {
			t.Fatalf("expected StoreError with op query, got %v", err)
		}
	}
	_ = mock.ExpectationsWereMet()
}

func TestQueryValidationError(t *testing.T) {
	s := NewStore(nil, Config{Dimension: 2})
	q := model.Query{TenantID: "", Namespace: "n", QueryVector: []float32{1}}
	if _, err := s.Query(context.Background(), q); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestQueryInvalidTableName(t *testing.T) {
	s := NewStore(nil, Config{TableName: "bad-name!", Dimension: 1})
	q := model.Query{TenantID: "t", Namespace: "n", QueryVector: []float32{1}}
	if _, err := s.Query(context.Background(), q); err == nil {
		t.Fatalf("expected invalid table name error")
	}
}

func TestPrometheusObserverMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	obs, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: reg})
	if err != nil {
		t.Fatalf("new observer: %v", err)
	}

	ctx, end := obs.Trace(context.Background(), "op")
	end(nil)
	obs.RecordLatency(ctx, "op", 10*time.Millisecond, errors.New("boom"))

	requests := testutil.ToFloat64(obs.requests.WithLabelValues("op", "ok"))
	if requests != 1 {
		t.Fatalf("expected 1 ok request, got %v", requests)
	}
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	foundLatency := false
	for _, mf := range mfs {
		if mf.GetName() != "platform_rag_pgvector_latency_seconds" {
			continue
		}
		for _, m := range mf.Metric {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "status" && lp.GetValue() == "error" && m.GetHistogram().GetSampleCount() > 0 {
					foundLatency = true
				}
			}
		}
	}
	if !foundLatency {
		t.Fatalf("expected latency with status=error")
	}
	if statusLabel(nil) != "ok" || statusLabel(errors.New("x")) != "error" {
		t.Fatalf("status label mismatch")
	}
}

func TestPrometheusObserverRegisterError(t *testing.T) {
	if _, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: failRegisterer{err: errors.New("reg fail")}}); err == nil {
		t.Fatalf("expected registration error")
	}
}

func TestPrometheusObserverAlreadyRegistered(t *testing.T) {
	reg := prometheus.NewRegistry()
	first, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: reg})
	if err != nil {
		t.Fatalf("first observer: %v", err)
	}
	second, err := NewPrometheusObserver(PrometheusObserverConfig{Registerer: reg})
	if err != nil {
		t.Fatalf("second observer: %v", err)
	}
	if first.requests != second.requests || first.latency != second.latency {
		t.Fatalf("expected existing collectors to be reused")
	}
}

func TestPrometheusObserverDefaultRegisterer(t *testing.T) {
	nameSuffix := time.Now().UnixNano()
	obs, err := NewPrometheusObserver(PrometheusObserverConfig{
		RequestsCounterOpts:  prometheus.CounterOpts{Name: fmt.Sprintf("pgvector_requests_%d", nameSuffix)},
		LatencyHistogramOpts: prometheus.HistogramOpts{Name: fmt.Sprintf("pgvector_latency_%d", nameSuffix)},
	})
	if err != nil {
		t.Fatalf("observer with default registerer: %v", err)
	}
	ctx, end := obs.Trace(context.Background(), "op")
	end(nil)
	obs.RecordLatency(ctx, "op", time.Millisecond, nil)
}

func TestPingSuccess(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.MonitorPingsOption(true))
	defer db.Close()
	s := NewStore(db, Config{})
	mock.ExpectPing()
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("ping success expected: %v", err)
	}
	_ = mock.ExpectationsWereMet()
}

func TestNoopObserverRecordLatency(t *testing.T) {
	var obs Observer = NoopObserver{}
	obs.RecordLatency(context.Background(), "op", time.Millisecond, nil)
}
