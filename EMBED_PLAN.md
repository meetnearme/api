# Create HTML Embed System for Organizations

## Overview

Create a JavaScript-based embed system that allows organizations with `subGrowth` role to embed their events section on external websites. The embed will extract the events section from `home.templ`, replace sidebar-dependent search with inline search variants, include Tailwind CSS, and handle HTMX/Alpine.js dependencies.

## Implementation Plan

### 1. Create Embed Endpoint (`/api/embed/events`)

   - **File**: `functions/gateway/handlers/partial_handlers.go`
   - **Function**: `GetEmbedEventsPartial`
   - **Auth**: `None` (public endpoint, but role-checked server-side)
   - **Query Params**: 
     - `ownerId` (required) - Zitadel userId
     - `searchStyle` (optional) - "inline" | "dialog" | "header" (default: "inline")
     - Standard search params: `q`, `radius`, `lat`, `lon`, `start_time`, `end_time`, etc.
   - **Logic**:
     - Use `helpers.GetUserRoles(ownerId)` to fetch roles
     - Check if user has `constants.Roles[constants.SubGrowth]` using `helpers.HasRequiredRole()`
     - Return 404 if role check fails
     - Fetch events using existing `GetSearchParamsFromReq()` and `SearchWeaviateEvents()`
     - Render embed template with `pageUser` set to the ownerId
   - **CORS**: Add CORS headers (`Access-Control-Allow-Origin: *`)

### 2. Create Embed Template

   - **File**: `functions/gateway/templates/partials/embed_events.templ`
   - **Content**: Extract events section from `home.templ` (lines 398-634)
   - **Changes**:
     - Remove sidebar/flyout dependencies (`@FilterButton()`, drawer clicks)
     - Remove "Add events via link" form (not needed in embed)
     - Create search bar variants based on `searchStyle` parameter:
       - **inline**: Full-width search bar (current style, no filter button)
       - **dialog**: Search button opens modal/dialog
       - **header**: Compact header bar style
     - Include Alpine.js state initialization (simplified version)
     - Include HTMX form setup
     - Scoped CSS classes (prefix with `mnm-embed-` to avoid conflicts)
   - **Parameters**: `events`, `pageUser`, `cfLocation`, `latStr`, `lonStr`, `searchStyle`, `roleClaims`, `userId`

### 3. Create JavaScript Embed Script

   - **File**: `static/assets/mnm-embed.js` (or serve from Go template)
   - **Functionality**:
     - Check if HTMX is loaded, load if not (`https://unpkg.com/htmx.org@1.9.12`)
     - Check if Alpine.js is loaded, load if not (`https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js`)
     - Load HTMX JSON extension if needed
     - Inject Tailwind CSS stylesheet (or reference existing if detected)
     - Fetch embed content from `/api/embed/events?ownerId={ownerId}&searchStyle={style}`
     - Inject HTML into target element
     - Initialize Alpine.js data attributes
   - **API**: `window.MeetNearMeEmbed.init({ ownerId, target, searchStyle, apiBaseUrl })`
   - **Options**:
     - `ownerId` (required)
     - `target` (optional, default: find element with `data-mnm-embed` attribute)
     - `searchStyle` (optional, default: "inline")
     - `apiBaseUrl` (optional, default: detect from script src or use `APEX_URL`)

### 4. Add CORS Support

   - **File**: `functions/gateway/main.go` or middleware
   - Add CORS middleware function that sets headers:
     - `Access-Control-Allow-Origin: *`
     - `Access-Control-Allow-Methods: GET, OPTIONS`
     - `Access-Control-Allow-Headers: Content-Type`
   - Apply to `/api/embed/*` routes

### 5. Add Route Registration

   - **File**: `functions/gateway/main.go`
   - Add to `InitRoutes()`:
     ```go
     {"/api/embed/events{trailingslash:\\/?}", "GET", handlers.GetEmbedEventsPartial, None},
     ```
   - Also add OPTIONS handler for CORS preflight

### 6. Extract Search Styles into Components

   - **File**: `functions/gateway/templates/partials/embed_search.templ`
   - Create separate templates for each search style:
     - `EmbedSearchInline()` - Current full-width search bar
     - `EmbedSearchDialog()` - Button opens modal with search form
     - `EmbedSearchHeader()` - Compact header bar style
   - Each handles Alpine.js state and HTMX form submission

### 7. Update EventsInner for Embed Context

   - **File**: `functions/gateway/templates/pages/home.templ`
   - Add optional `embedMode` parameter to `EventsInner()` template
   - When `embedMode=true`:
     - Remove admin/export buttons
     - Use relative URLs for event links (add `data-mnm-base-url` attribute)
     - Simplify event display (remove some admin features)

### 8. Create Embed Documentation

   - **File**: `EMBED.md`
   - Document embed code usage, parameters, styling options
   - Include examples for each search style variant

## Technical Considerations

1. **CSS Scoping**: Use Tailwind's `important` strategy or CSS variable overrides to allow host page customization
2. **Event Links**: Use `data-mnm-base-url` attribute to construct full URLs for event detail pages
3. **Alpine.js Isolation**: Prefix Alpine stores with `mnmEmbed_` to avoid conflicts
4. **HTMX Configuration**: Use `hx-ext="json-enc"` and ensure proper endpoint targeting
5. **Role Checking**: Use `helpers.GetUserRoles()` + `helpers.HasRequiredRole()` for server-side validation
6. **Error Handling**: Return 404 with clear message if role check fails or ownerId invalid

## Files to Create/Modify

**New Files**:

- `functions/gateway/templates/partials/embed_events.templ`
- `functions/gateway/templates/partials/embed_search.templ`
- `static/assets/mnm-embed.js`
- `EMBED.md`

**Modified Files**:

- `functions/gateway/handlers/partial_handlers.go` - Add `GetEmbedEventsPartial` handler
- `functions/gateway/main.go` - Add embed route, CORS middleware
- `functions/gateway/templates/pages/home.templ` - Update `EventsInner` for embed mode
- `functions/gateway/helpers/utils.go` - Possibly add helper for role checking by userId

## Testing Considerations

- Test role check with valid/invalid ownerIds
- Test all three search style variants
- Test CORS from different origins
- Test HTMX/Alpine.js initialization when already loaded vs. fresh load
- Test CSS conflicts with common frameworks (Bootstrap, Tailwind, etc.)
- Test event link generation with different base URLs

## Implementation Todos

- [ ] Create GetEmbedEventsPartial handler in partial_handlers.go with role check using helpers.GetUserRoles() and helpers.HasRequiredRole()
- [ ] Add CORS middleware to main.go for /api/embed/* routes with appropriate headers
- [ ] Create embed_events.templ template extracting events section from home.templ, removing sidebar dependencies
- [ ] Create embed_search.templ with three search style variants: inline, dialog, header
- [ ] Update EventsInner template in home.templ to support embedMode parameter for simplified display
- [ ] Create mnm-embed.js script that loads HTMX/Alpine.js, fetches embed content, and injects into target element
- [ ] Add /api/embed/events route to main.go InitRoutes() with OPTIONS handler for CORS preflight
- [ ] Create EMBED.md documentation with usage examples and styling options

