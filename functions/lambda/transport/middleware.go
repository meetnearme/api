package transport

import (
	"context"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/meetnearme/api/functions/lambda/helpers"
)

func ParseCookies(ctx context.Context, r Request) (context.Context, Request, *HTTPError) {
	cookies := make(map[string]string)
	for _, cookie := range r.Cookies {
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			cookies[parts[0]] = parts[1]
		}
	}

	ctx = context.WithValue(ctx, "cookiesMap", cookies)
	return ctx, r, nil
}

func RequireHeaderAuthorization(ctx context.Context, r Request) (context.Context, Request, *HTTPError) {
	cookiesMap := ctx.Value("cookiesMap").(map[string]string)
	sessionToken := cookiesMap[helpers.SESSION_COOKIE]
	if sessionToken == "" {
		httpError := &HTTPError{
			Status:          302,
			Message:         "Unauthorized. Session token missing.",
			ErrorComponent:  nil,
			ResponseHeaders: map[string]string{"Location": os.Getenv("APEX_URL") + "/login?redirect=" + r.RawPath},
		}
		return ctx, r, httpError
	}

	_, err := jwt.Decode(ctx, &jwt.DecodeParams{Token: sessionToken})
	if err != nil {
		httpError := &HTTPError{
			Status:         403,
			Message:        "Forbidden. Failed to decode session token.",
			ErrorComponent: nil,
		}
		return ctx, r, httpError
	}

	params := &http.AuthorizationParams{}
	params.Token = sessionToken

	claims, err := jwt.Verify(ctx, &params.VerifyParams)
	if err != nil {
		httpError := &HTTPError{
			Status:         403,
			Message:        "Forbidden. Invalid session token.",
			ErrorComponent: nil,
		}
		return ctx, r, httpError
	}

	newCtx := clerk.ContextWithSessionClaims(ctx, claims)
	return newCtx, r, nil
}
