package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

// CreateCustomerPortalSession handles requests to create a Stripe customer portal session
// This redirects the user to Stripe's customer portal for managing their subscription
func CreateCustomerPortalSession(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()

	// Authenticate user and get Zitadel ID
	userInfo := constants.UserInfo{}
	if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(constants.UserInfo)
	}
	zitadelUserID := userInfo.Sub
	if zitadelUserID == "" {
		transport.SendServerRes(w, []byte("Missing user ID"), http.StatusUnauthorized, nil)
		return nil
	}

	// Get return URL from query parameter, default to /admin/subscriptions
	returnURL := r.URL.Query().Get("return_url")
	if returnURL == "" {
		returnURL = os.Getenv("APEX_URL") + "/admin/subscriptions"
	} else {
		// Ensure return URL is absolute
		if len(returnURL) > 0 && returnURL[0] != '/' && !isAbsoluteURL(returnURL) {
			returnURL = os.Getenv("APEX_URL") + returnURL
		}
	}

	// Get optional subscription ID and flow type for deep linking
	// If subscription_id is provided, creates a deep link to manage that specific subscription
	subscriptionID := r.URL.Query().Get("subscription_id")
	flowType := r.URL.Query().Get("flow_type")
	// Valid flow types: constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_CANCEL, STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE,
	// STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE_CONFIRM, or STRIPE_PORTAL_FLOW_PAYMENT_METHOD_UPDATE
	// If subscription_id is provided but flow_type is empty, defaults to STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE
	if subscriptionID != "" && flowType == "" {
		flowType = constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE
	}

	// Get subscription service
	subscriptionService := services.NewStripeSubscriptionService()

	// Find or create Stripe customer using GetOrCreateCustomerByExternalID
	// This searches by zitadel_user_id metadata in Stripe (one-way lookup)
	stripeCustomer, err := subscriptionService.GetOrCreateCustomerByExternalID(
		zitadelUserID,
		userInfo.Email,
		userInfo.Name,
	)
	if err != nil {
		log.Printf("Error finding or creating Stripe customer for Zitadel user %s: %v", zitadelUserID, err)
		transport.SendServerRes(w, []byte("Failed to find or create customer: "+err.Error()), http.StatusInternalServerError, err)
		return err
	}

	if stripeCustomer == nil {
		log.Printf("Unexpected: GetOrCreateCustomerByExternalID returned nil customer for Zitadel user %s", zitadelUserID)
		transport.SendServerRes(w, []byte("Failed to create customer"), http.StatusInternalServerError, nil)
		return nil
	}

	log.Printf("Found/created Stripe customer %s for Zitadel user %s", stripeCustomer.ID, zitadelUserID)

	// Generate portal session URL using CreateCustomerPortalSession from subscription service
	// If subscriptionID is provided, creates a deep link to that specific subscription
	portalSession, err := subscriptionService.CreateCustomerPortalSession(stripeCustomer.ID, returnURL, subscriptionID, flowType)
	if err != nil {
		log.Printf("Error creating customer portal session for customer %s: %v", stripeCustomer.ID, err)
		transport.SendServerRes(w, []byte("Failed to create portal session: "+err.Error()), http.StatusInternalServerError, err)
		return err
	}

	log.Printf("Created customer portal session %s for customer %s, redirecting to %s", portalSession.ID, stripeCustomer.ID, portalSession.URL)

	// Redirect user to Stripe customer portal for management actions
	http.Redirect(w, r, portalSession.URL, http.StatusSeeOther)
	return nil
}

// CreateCustomerPortalSessionHandler wraps CreateCustomerPortalSession for route registration
func CreateCustomerPortalSessionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		CreateCustomerPortalSession(w, r)
	}
}

// isAbsoluteURL checks if a URL is absolute (starts with http:// or https://)
func isAbsoluteURL(url string) bool {
	if len(url) < 8 {
		return false
	}
	return url[0:7] == "http://" || (len(url) >= 8 && url[0:8] == "https://")
}
