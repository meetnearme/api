package services

import (
    "context"
    "net/http"
    "github.com/zitadel/oidc/v3/pkg/oidc"
    openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"
)

type Interceptor interface {
    CheckAuthentication() func(http.Handler) http.Handler
    RequireAuthentication() func(http.Handler) http.Handler
    Context(ctx context.Context) *openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]
}

type Authenticator interface {
   Authenticate(w http.ResponseWriter, r *http.Request, redirectURL string)
   Callback(w http.ResponseWriter, r *http.Request)
   Logout(w http.ResponseWriter, r *http.Request)
}
