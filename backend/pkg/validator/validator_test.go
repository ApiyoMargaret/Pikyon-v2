package validator_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/validator"
)

func init() { gin.SetMode(gin.TestMode) }

type createReq struct {
	Title      string   `json:"title" binding:"required,min=1,max=10"`
	Email      string   `json:"email" binding:"omitempty,email"`
	Visibility string   `json:"visibility" binding:"omitempty,oneof=private public"`
	MemoryID   string   `json:"memory_id" binding:"omitempty,uuid"`
	Date       string   `json:"memory_date" binding:"omitempty,datetime=2006-01-02"`
	Tags       []string `json:"tags" binding:"omitempty,max=2"`
	Count      int      `json:"count" binding:"omitempty,min=1,max=5"`
	Story      *string  `json:"story"`
	Password   string   `json:"password" binding:"omitempty,min=8"`
}

type nestedReq struct {
	Items []struct {
		Name string `json:"name" binding:"required"`
	} `json:"items" binding:"dive"`
}

type listQuery struct {
	Limit      int    `form:"limit" binding:"omitempty,min=1,max=50"`
	Visibility string `form:"visibility" binding:"omitempty,oneof=private public"`
	FromDate   string `form:"from_date" binding:"omitempty,datetime=2006-01-02"`
}

type errBody struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"details"`
	} `json:"error"`
}

// post sends body to a handler that calls Bind and reports whether Bind returned true.
func post(t *testing.T, body string, obj any) (*httptest.ResponseRecorder, bool, bool) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	ok := validator.Bind(c, obj)
	return w, ok, c.IsAborted()
}

func details(t *testing.T, w *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var b errBody
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, w.Body.String())
	}
	if b.Success || b.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("want a VALIDATION_ERROR, got %s", w.Body.String())
	}
	m := map[string]string{}
	for _, d := range b.Error.Details {
		m[d.Field] = d.Message
	}
	return m
}

func TestBind_ValidBodyPasses(t *testing.T) {
	var req createReq
	w, ok, aborted := post(t, `{"title":"Beach","email":"a@b.co","tags":["x"],"count":3,"story":"hi"}`, &req)
	if !ok || aborted || w.Body.Len() != 0 {
		t.Fatalf("valid body must pass without writing a response: ok=%v aborted=%v body=%q", ok, aborted, w.Body.String())
	}
	if req.Title != "Beach" || req.Count != 3 || req.Story == nil || *req.Story != "hi" {
		t.Errorf("struct not filled: %+v", req)
	}
}

func TestBind_FailureWritesEnvelopeAndAborts(t *testing.T) {
	var req createReq
	w, ok, aborted := post(t, `{}`, &req)
	if ok || !aborted || w.Code != http.StatusBadRequest {
		t.Fatalf("ok=%v aborted=%v status=%d", ok, aborted, w.Code)
	}
	if d := details(t, w); d["title"] != "Title is required" {
		t.Errorf("details = %v", d)
	}
}

func TestBind_FieldMessages(t *testing.T) {
	tests := []struct {
		name, body, field, want string
	}{
		{"required", `{}`, "title", "Title is required"},
		{"max length", `{"title":"abcdefghijk"}`, "title", "Title must be at most 10 characters"},
		{"email", `{"title":"a","email":"nope"}`, "email", "Email must be a valid email address"},
		{"oneof", `{"title":"a","visibility":"secret"}`, "visibility", "Visibility must be one of: private, public"},
		{"uuid", `{"title":"a","memory_id":"123"}`, "memory_id", "Memory id must be a valid UUID"},
		{"date format", `{"title":"a","memory_date":"28/10/2026"}`, "memory_date", "Memory date must be a date in the format YYYY-MM-DD"},
		{"list too long", `{"title":"a","tags":["1","2","3"]}`, "tags", "Tags must be at most 2 items"},
		{"number too small", `{"title":"a","count":0,"password":"x"}`, "password", "Password must be at least 8 characters"},
		{"number too big", `{"title":"a","count":9}`, "count", "Count must be at most 5"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req createReq
			w, ok, _ := post(t, tc.body, &req)
			if ok {
				t.Fatal("expected validation to fail")
			}
			if got := details(t, w)[tc.field]; got != tc.want {
				t.Errorf("%s: message = %q, want %q", tc.field, got, tc.want)
			}
		})
	}
}

func TestBind_ReportsAllFieldsAtOnce(t *testing.T) {
	var req createReq
	w, _, _ := post(t, `{"email":"bad","visibility":"x"}`, &req)
	d := details(t, w)
	for _, f := range []string{"title", "email", "visibility"} {
		if d[f] == "" {
			t.Errorf("missing detail for %q in %v", f, d)
		}
	}
}

func TestBind_UsesJSONNamesNotGoNames(t *testing.T) {
	var req createReq
	w, _, _ := post(t, `{}`, &req)
	for field := range details(t, w) {
		if field == "Title" || strings.Contains(field, "createReq") {
			t.Errorf("detail field %q leaks Go names", field)
		}
	}
}

func TestBind_NestedFieldPath(t *testing.T) {
	var req nestedReq
	w, _, _ := post(t, `{"items":[{"name":"ok"},{}]}`, &req)
	if got := details(t, w)["items[1].name"]; got != "Name is required" {
		t.Errorf("details = %v", details(t, w))
	}
}

func TestBind_BodyProblems(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"empty body", ``, "Request body is required"},
		{"malformed JSON", `{"title": `, "Request body must be valid JSON"},
		{"not JSON at all", `hello`, "Request body must be valid JSON"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req createReq
			w, ok, _ := post(t, tc.body, &req)
			if ok {
				t.Fatal("expected failure")
			}
			if got := details(t, w)["body"]; got != tc.want {
				t.Errorf("message = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBind_WrongJSONTypes(t *testing.T) {
	tests := []struct{ body, field, want string }{
		{`{"title":5}`, "title", "Title must be text"},
		{`{"title":"a","count":"many"}`, "count", "Count must be a whole number"},
		{`{"title":"a","tags":"x"}`, "tags", "Tags must be a list"},
		{`{"title":"a","story":5}`, "story", "Story must be text"},
	}
	for _, tc := range tests {
		var req createReq
		w, ok, _ := post(t, tc.body, &req)
		if ok {
			t.Fatalf("%s: expected failure", tc.body)
		}
		if got := details(t, w)[tc.field]; got != tc.want {
			t.Errorf("%s: %q, want %q", tc.body, got, tc.want)
		}
	}
}

func TestBind_OversizedBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"title":"`+strings.Repeat("a", 5000)+`"}`))
	c.Request.Body = http.MaxBytesReader(w, c.Request.Body, 100)
	var req createReq
	if validator.Bind(c, &req) {
		t.Fatal("expected failure")
	}
	if got := details(t, w)["body"]; got != "Request body is too large" {
		t.Errorf("message = %q", got)
	}
}

// Submitted values must never be echoed back in error messages.
func TestBind_NeverEchoesSubmittedValues(t *testing.T) {
	var req createReq
	w, _, _ := post(t, `{"title":"a","email":"hunter2-secret","password":"short1","count":"hunter2-secret"}`, &req)
	if strings.Contains(w.Body.String(), "hunter2-secret") || strings.Contains(w.Body.String(), "short1") {
		t.Errorf("response echoes a submitted value: %s", w.Body.String())
	}
}

func TestBind_ProgrammerErrorIs500(t *testing.T) {
	var req createReq
	w, ok, _ := post(t, `{"title":"a"}`, req) // non-pointer: a bug in the calling code
	if ok || w.Code != http.StatusInternalServerError {
		t.Fatalf("ok=%v status=%d, want 500", ok, w.Code)
	}
	if strings.Contains(w.Body.String(), "Unmarshal") {
		t.Errorf("internal detail leaked: %s", w.Body.String())
	}
}

func TestBindQuery(t *testing.T) {
	get := func(rawQuery string) (*httptest.ResponseRecorder, bool, listQuery) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
		var q listQuery
		return w, validator.BindQuery(c, &q), q
	}

	if _, ok, q := get("limit=20&visibility=private&from_date=2026-10-28"); !ok || q.Limit != 20 || q.Visibility != "private" {
		t.Errorf("valid query rejected: ok=%v q=%+v", ok, q)
	}

	w, ok, _ := get("limit=99&visibility=secret&from_date=yesterday")
	if ok {
		t.Fatal("expected failure")
	}
	d := details(t, w)
	if d["limit"] != "Limit must be at most 50" || d["visibility"] != "Visibility must be one of: private, public" ||
		d["from_date"] != "From date must be a date in the format YYYY-MM-DD" {
		t.Errorf("details use form names? %v", d)
	}

	w, ok, _ = get("limit=abc")
	if ok {
		t.Fatal("expected failure for a non-numeric limit")
	}
	if strings.Contains(w.Body.String(), "abc") {
		t.Errorf("response echoes the submitted value: %s", w.Body.String())
	}
	details(t, w) // still a VALIDATION_ERROR envelope
}

func TestTranslate_ReturnsValidationError(t *testing.T) {
	e := validator.Translate(nil)
	if e == nil || e.Code != "VALIDATION_ERROR" {
		t.Errorf("Translate(nil) = %+v", e) // unknown/empty errors degrade to a safe validation error
	}
}

type extraReq struct {
	Code    string   `json:"code" binding:"omitempty,len=4"`
	Site    string   `json:"site" binding:"omitempty,url"`
	Handle  string   `json:"handle" binding:"omitempty,alphanum"`
	Stamp   string   `json:"stamp" binding:"omitempty,datetime=2006-01-02T15:04"`
	Longer  string   `json:"longer" binding:"omitempty,gt=3"`
	Shorter []string `json:"shorter" binding:"omitempty,lt=2"`
	Rating  int      `json:"rating" binding:"omitempty,gte=1"`
	Cap     int      `json:"cap" binding:"omitempty,lte=10"`
	Flag    bool     `json:"flag"`
	Ratio   float64  `json:"ratio"`
	Meta    struct{} `json:"meta"`
	ID4     string   `json:"id4" binding:"omitempty,uuid4"`
	Same    string   `json:"same" binding:"omitempty,startswith=ab"`
}

func TestBind_MoreMessages(t *testing.T) {
	tests := []struct{ body, field, want string }{
		{`{"code":"abc"}`, "code", "Code must be exactly 4 characters"},
		{`{"site":"not a url"}`, "site", "Site must be a valid URL"},
		{`{"handle":"a b!"}`, "handle", "Handle must contain only letters and numbers"},
		{`{"stamp":"2026-10-28"}`, "stamp", "Stamp must match the format 2006-01-02T15:04"},
		{`{"longer":"ab"}`, "longer", "Longer must be greater than 3 characters"},
		{`{"shorter":["a","b"]}`, "shorter", "Shorter must be less than 2 items"},
		{`{"rating":0,"cap":11}`, "cap", "Cap must be at most 10"},
		{`{"id4":"x"}`, "id4", "Id4 must be a valid UUID"},
		{`{"same":"zz"}`, "same", "Same is invalid"},
		{`{"flag":"yes"}`, "flag", "Flag must be true or false"},
		{`{"ratio":"half"}`, "ratio", "Ratio must be a number"},
		{`{"meta":"x"}`, "meta", "Meta must be an object"},
	}
	for _, tc := range tests {
		var req extraReq
		w, ok, _ := post(t, tc.body, &req)
		if ok {
			t.Errorf("%s: expected failure", tc.body)
			continue
		}
		if got := details(t, w)[tc.field]; got != tc.want {
			t.Errorf("%s: message = %q, want %q", tc.body, got, tc.want)
		}
	}
}

func TestBind_SingularUnits(t *testing.T) {
	type one struct {
		S string   `json:"s" binding:"max=1"`
		L []string `json:"l" binding:"max=1"`
	}
	var o one
	w, _, _ := post(t, `{"s":"ab","l":["a","b"]}`, &o)
	d := details(t, w)
	if d["s"] != "S must be at most 1 character" || d["l"] != "L must be at most 1 item" {
		t.Errorf("singular wording wrong: %v", d)
	}
}
