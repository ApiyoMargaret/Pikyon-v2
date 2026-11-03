package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ApiyoMargaret/Pikyon-v2/backend/internal/auth"
	"github.com/ApiyoMargaret/Pikyon-v2/backend/pkg/validator"
)

func init() { gin.SetMode(gin.TestMode) }

// bind runs the real request validator over body and returns the field messages (empty = valid).
func bind(t *testing.T, body string, obj any) map[string]string {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if validator.Bind(c, obj) {
		return nil
	}
	var resp struct {
		Error struct {
			Details []struct{ Field, Message string } `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad error body: %s", w.Body.String())
	}
	out := map[string]string{}
	for _, d := range resp.Error.Details {
		out[d.Field] = d.Message
	}
	return out
}

func TestRegisterRequest_Valid(t *testing.T) {
	var r auth.RegisterRequest
	if errs := bind(t, `{"email":"jane@example.com","password":"correct horse","name":"Jane"}`, &r); errs != nil {
		t.Fatalf("valid payload rejected: %v", errs)
	}
	if r.Email != "jane@example.com" || r.Name != "Jane" {
		t.Errorf("fields not filled: %+v", r)
	}
}

func TestRegisterRequest_Messages(t *testing.T) {
	tests := []struct{ name, body, field, want string }{
		{"missing everything", `{}`, "email", "Email is required"},
		{"missing password", `{"email":"a@b.co","name":"J"}`, "password", "Password is required"},
		{"missing name", `{"email":"a@b.co","password":"12345678"}`, "name", "Name is required"},
		{"bad email", `{"email":"nope","password":"12345678","name":"J"}`, "email", "Email must be a valid email address"},
		{"short password", `{"email":"a@b.co","password":"1234567","name":"J"}`, "password", "Password must be at least 8 characters"},
		{"long password", `{"email":"a@b.co","password":"` + strings.Repeat("a", 73) + `","name":"J"}`, "password", "Password must be at most 72 characters"},
		{"long name", `{"email":"a@b.co","password":"12345678","name":"` + strings.Repeat("a", 101) + `"}`, "name", "Name must be at most 100 characters"},
		{"long email", `{"email":"` + strings.Repeat("a", 251) + `@b.co","password":"12345678","name":"J"}`, "email", "Email must be at most 255 characters"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r auth.RegisterRequest
			errs := bind(t, tc.body, &r)
			if errs == nil {
				t.Fatal("expected validation to fail")
			}
			if errs[tc.field] != tc.want {
				t.Errorf("%s: %q, want %q (all: %v)", tc.field, errs[tc.field], tc.want, errs)
			}
		})
	}
}

// Field names in errors are the JSON names the client sent: "name", not the old "full_name".
func TestRegisterRequest_UsesNameNotFullName(t *testing.T) {
	var r auth.RegisterRequest
	errs := bind(t, `{"email":"a@b.co","password":"12345678","full_name":"Jane"}`, &r)
	if _, ok := errs["name"]; !ok {
		t.Errorf("a payload with the old full_name field must fail on name: %v", errs)
	}
}

func TestLoginRequest(t *testing.T) {
	var ok auth.LoginRequest
	if errs := bind(t, `{"email":"jane@example.com","password":"x"}`, &ok); errs != nil {
		t.Errorf("valid login rejected: %v", errs)
	}
	var bad auth.LoginRequest
	errs := bind(t, `{"email":"nope","password":""}`, &bad)
	if errs["email"] != "Email must be a valid email address" || errs["password"] != "Password is required" {
		t.Errorf("errors = %v", errs)
	}
	var long auth.LoginRequest
	if errs := bind(t, `{"email":"a@b.co","password":"`+strings.Repeat("a", 73)+`"}`, &long); errs["password"] == "" {
		t.Error("an over-long password must be rejected before it reaches bcrypt")
	}
}

// The token may come from the header instead, so an empty body field must be accepted here.
func TestRefreshRequest_TokenIsOptionalInBody(t *testing.T) {
	var r auth.RefreshRequest
	if errs := bind(t, `{}`, &r); errs != nil {
		t.Errorf("an empty body must be accepted (the header may carry the token): %v", errs)
	}
	var big auth.RefreshRequest
	if errs := bind(t, `{"refresh_token":"`+strings.Repeat("a", 513)+`"}`, &big); errs["refresh_token"] == "" {
		t.Error("an absurdly long token must be rejected")
	}
}

// The public view of a user must never carry anything secret, even if the model holds it.
func TestUserResponse_NeverExposesSecrets(t *testing.T) {
	hash := "$2a$12$secretsecretsecretsecretsecretsecretsecretsecretsecretse"
	avatar := "https://example.com/a.png"
	u := &auth.User{
		ID: "11111111-1111-1111-1111-111111111111", Email: "jane@example.com", EmailVerified: true,
		Name: "Jane", AvatarURL: &avatar, PasswordHash: &hash, IsActive: true,
		PasswordChangedAt: time.Now(), CreatedAt: time.Now(),
	}
	raw, err := json.Marshal(u.ToResponse())
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, forbidden := range []string{"secretsecret", "password", "hash", "is_active", "totp", "pin"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Errorf("public user JSON contains %q: %s", forbidden, body)
		}
	}
	var back map[string]any
	_ = json.Unmarshal(raw, &back)
	for _, want := range []string{"id", "email", "email_verified", "name", "avatar_url", "created_at"} {
		if _, ok := back[want]; !ok {
			t.Errorf("public user JSON is missing %q", want)
		}
	}
}

func TestUserResponse_NullAvatarIsExplicitNull(t *testing.T) {
	raw, _ := json.Marshal((&auth.User{ID: "x", Email: "a@b.co"}).ToResponse())
	if !strings.Contains(string(raw), `"avatar_url":null`) {
		t.Errorf("avatar_url must be null, not omitted: %s", raw)
	}
}
