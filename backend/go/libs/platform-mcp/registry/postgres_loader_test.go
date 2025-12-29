package registry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestPostgresLoaderListTools(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	updated := time.Now().UTC()
	rows := mock.NewRows([]string{"name", "version", "display_name", "summary", "description", "input_schema", "output_schema", "scopes", "tags", "allow_hosts", "idempotency_key", "max_body_bytes", "max_duration_ms", "deprecated", "etag", "updated_at"}).
		AddRow("echo", "1.0.0", "Echo", "sum", "desc", []byte(`{}`), []byte(`{}`), []string{"s1"}, []string{"tag"}, []string{"host"}, "idem", int64(123), int64(5000), false, "e1", updated)
	mock.ExpectQuery("FROM tools").WithArgs("t1").WillReturnRows(rows)

	loader := NewPostgresLoader(mock)
	tools, err := loader.ListTools(context.Background(), "t1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tools) != 1 || tools[0].TenantID != "t1" || tools[0].MaxDuration != 5*time.Second {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestPostgresLoaderDescribeTool(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	updated := time.Now().UTC()
	mock.ExpectQuery("FROM tools").WithArgs("t1", "echo").
		WillReturnRows(mock.NewRows([]string{"name", "version", "display_name", "summary", "description", "input_schema", "output_schema", "scopes", "tags", "allow_hosts", "idempotency_key", "max_body_bytes", "max_duration_ms", "deprecated", "etag", "updated_at"}).
			AddRow("echo", "1.0.0", "Echo", "sum", "desc", []byte(`{}`), []byte(`{}`), []string{"s1"}, []string{"tag"}, []string{"host"}, "idem", int64(123), int64(0), false, "e1", updated))

	loader := NewPostgresLoader(mock)
	tool, err := loader.DescribeTool(context.Background(), "t1", "echo")
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if tool.Name != "echo" || tool.TenantID != "t1" || tool.MaxDuration != 0 {
		t.Fatalf("unexpected tool: %+v", tool)
	}
}

func TestPostgresLoaderListError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("FROM tools").WithArgs("t1").WillReturnError(context.DeadlineExceeded)
	loader := NewPostgresLoader(mock)
	if _, err := loader.ListTools(context.Background(), "t1"); err == nil {
		t.Fatalf("expected error from list query")
	}
}

func TestPostgresLoaderDescribeError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("FROM tools").WithArgs("t1", "missing").WillReturnError(context.Canceled)
	loader := NewPostgresLoader(mock)
	if _, err := loader.DescribeTool(context.Background(), "t1", "missing"); err == nil {
		t.Fatalf("expected describe error")
	}
}

func TestPostgresLoaderListRowsError(t *testing.T) {
	loader := NewPostgresLoader(stubPool{rows: &fakeRows{err: errors.New("rows err")}})

	if _, err := loader.ListTools(context.Background(), "t1"); err == nil {
		t.Fatalf("expected rows error")
	}
}

func TestPostgresLoaderListScanError(t *testing.T) {
	loader := NewPostgresLoader(stubPool{rows: &fakeRows{scans: []func(dest ...any) error{func(dest ...any) error {
		return errors.New("scan fail")
	}}}})

	if _, err := loader.ListTools(context.Background(), "t1"); err == nil {
		t.Fatalf("expected scan error")
	}
}

func TestPostgresLoaderDescribeScanError(t *testing.T) {
	loader := NewPostgresLoader(stubPool{row: stubRow{err: errors.New("scan fail")}})

	if _, err := loader.DescribeTool(context.Background(), "t1", "echo"); err == nil {
		t.Fatalf("expected scan error")
	}
}

func TestPostgresLoaderDescribeSetsMaxDuration(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	updated := time.Now().UTC()
	mock.ExpectQuery("FROM tools").WithArgs("t1", "echo").
		WillReturnRows(mock.NewRows([]string{"name", "version", "display_name", "summary", "description", "input_schema", "output_schema", "scopes", "tags", "allow_hosts", "idempotency_key", "max_body_bytes", "max_duration_ms", "deprecated", "etag", "updated_at"}).
			AddRow("echo", "1.0.0", "Echo", "sum", "desc", []byte(`{}`), []byte(`{}`), []string{"s1"}, []string{"tag"}, []string{"host"}, "idem", int64(123), int64(1500), false, "e1", updated))

	loader := NewPostgresLoader(mock)
	tool, err := loader.DescribeTool(context.Background(), "t1", "echo")
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if tool.MaxDuration != 1500*time.Millisecond {
		t.Fatalf("expected max duration set, got %v", tool.MaxDuration)
	}
}

func TestRowsToTools(t *testing.T) {
	rows := &fakeRows{scans: []func(dest ...any) error{
		func(dest ...any) error {
			*(dest[0].(*string)) = "echo"
			*(dest[1].(*string)) = "1.0.0"
			*(dest[2].(*string)) = "Echo"
			*(dest[3].(*string)) = "sum"
			*(dest[4].(*string)) = "desc"
			*(dest[5].(*json.RawMessage)) = json.RawMessage(`{}`)
			*(dest[6].(*json.RawMessage)) = json.RawMessage(`{}`)
			*(dest[7].(*[]string)) = []string{"s1"}
			*(dest[8].(*[]string)) = []string{"tag"}
			*(dest[9].(*[]string)) = []string{"host"}
			*(dest[10].(*string)) = "idem"
			*(dest[11].(*int64)) = 123
			*(dest[12].(*int64)) = 500
			*(dest[13].(*bool)) = false
			*(dest[14].(*string)) = "e1"
			*(dest[15].(*time.Time)) = time.Now()
			return nil
		},
	}}

	tools, err := RowsToTools(rows, "t1")
	if err != nil {
		t.Fatalf("rows to tools: %v", err)
	}
	if len(tools) != 1 || tools[0].TenantID != "t1" {
		t.Fatalf("unexpected tools: %+v", tools)
	}

	badRows := &fakeRows{scans: []func(dest ...any) error{func(dest ...any) error { return errors.New("scan fail") }}}

	if _, err := RowsToTools(badRows, "t1"); err == nil {
		t.Fatalf("expected scan error")
	}

	errRows := &fakeRows{err: errors.New("rows err")}

	if _, err := RowsToTools(errRows, "t1"); err == nil {
		t.Fatalf("expected rows err")
	}
}

type fakeRows struct {
	idx   int
	scans []func(dest ...any) error
	err   error
}

func (f *fakeRows) Next() bool {
	if f.idx >= len(f.scans) {
		return false
	}
	f.idx++
	return true
}

func (f *fakeRows) Scan(dest ...any) error {
	return f.scans[f.idx-1](dest...)
}

func (f *fakeRows) Err() error                                   { return f.err }
func (f *fakeRows) Close()                                       {}
func (f *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (f *fakeRows) RawValues() [][]byte                          { return nil }
func (f *fakeRows) Conn() *pgx.Conn                              { return nil }

type stubPool struct {
	rows     pgx.Rows
	row      pgx.Row
	queryErr error
}

func (s stubPool) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	return s.rows, nil
}

func (s stubPool) QueryRow(context.Context, string, ...any) pgx.Row {
	if s.row != nil {
		return s.row
	}
	return stubRow{err: errors.New("no row")}
}

type stubRow struct{ err error }

func (s stubRow) Scan(dest ...any) error { return s.err }
