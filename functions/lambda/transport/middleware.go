package transport

import (
	"context"
	"errors"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/meetnearme/api/functions/lambda/helpers"
)

func ParseCookies(ctx context.Context, r Request) (context.Context, Request, error) {
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

func RequireHeaderAuthorization(ctx context.Context, r Request) (context.Context, Request, error) {
	cookiesMap := ctx.Value("cookiesMap").(map[string]string)
	sessionToken := cookiesMap[helpers.SESSION_COOKIE]
	if sessionToken == "" {
		return ctx, r, errors.New("Unauthorized")
	}

	_, err := jwt.Decode(ctx, &jwt.DecodeParams{Token: sessionToken})
	if err != nil {
		return ctx, r, err
	}

	params := &http.AuthorizationParams{}
	params.Token = sessionToken

	claims, err := jwt.Verify(ctx, &params.VerifyParams)
	if err != nil {
		return ctx, r, err
	}

	newCtx := clerk.ContextWithSessionClaims(ctx, claims)
	return newCtx, r, nil
}
