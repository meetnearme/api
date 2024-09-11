package services

import (
	"context"
	"net/http"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"
)

type MockAuthenticator struct{}

func (m *MockAuthenticator) Authenticate(w http.ResponseWriter, r *http.Request, redirectURL string) {
    // Do nothing in the mock
}

func (m *MockAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
    // Do nothing in the mock
}

func (m *MockAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
    // Do nothing in the mock
}

type MockInterceptor struct{}

func (m *MockInterceptor) CheckAuthentication() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            next.ServeHTTP(w, r)
        })
    }
}

func (m *MockInterceptor) RequireAuthentication() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            next.ServeHTTP(w, r)
        })
    }
}

func (m *MockInterceptor) Context(ctx context.Context) *openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo] {
    return &openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]{
       UserInfo: &oidc.UserInfo{
           Subject: "test-subject",
       },
    }
}

