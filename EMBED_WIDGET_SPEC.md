# Event List Embed Widget - High Level Specification

## Overview

A self-contained, embeddable widget that allows users to embed MeetNearMe's
event list with search and filter functionality on their own websites. The
widget will be injected via a single script tag (similar to iframe behavior but
without iframe isolation).

## Core Requirements

### 1. Embed Mechanism

- **Single script tag**: Users add one `<script>` tag to their HTML
- **Self-contained**: All HTML, CSS, and JavaScript bundled/injected by the
  script
- **No iframe**: Direct DOM injection to allow external CSS styling
- **Namespace isolation**: Use prefixed class names/IDs to avoid conflicts with
  host site CSS

### 2. Functionality to Extract

#### From Home Page (`home.templ`)

- **Search bar**: Text input with search button (lines ~567-585)
- **Search examples**: Quick search suggestion buttons (lines ~586-600)
- **Event list rendering**: `EventsInner` component for displaying events
- **Time filters**: "This Month", "Today", "Tomorrow", "This Week" buttons
  (lines ~697-730)
- **Alpine.js state management**: `urlState` store and `getHomeState()` function
- **HTMX form submission**: Event search form that calls `/api/html/events`
  endpoint

#### From Navbar (`navbar.templ`)

- **Filter sidebar**: Filter form UI (lines ~351-396)
- **Filter state management**: `getFilterFormState()` Alpine.js function
- **Filter submission**: `handleFilterSubmit()` function
- **Filter components**: Category, location, radius, date range filters

### 3. Conditional Logic Strategy

- **Add `isEmbed` boolean parameter** to existing templates (`Home()`,
  `Navbar()`, etc.)
- **Conditionally show/hide components** based on `isEmbed` flag (keep existing
  components, just hide when needed)
- **Create new `widget.templ` component** in `templates/components/` that
  composes:
  - Event search bar (from `home.templ`)
  - Event filters (from `navbar.templ`)
  - Event list rendering (using existing `EventsInner` component)
- **Skip when `isEmbed=true`**: navbar, footer, user auth UI, export buttons,
  share buttons, carousel controls
- **Include when `isEmbed=true`**: search bar, filters, event list, time filter
  buttons

### 4. Styling Approach

- **Minimal inline styles**: Only essential layout/functionality styles
- **Class-based styling**: Use semantic class names with embed prefix (e.g.,
  `mnm-embed-search`, `mnm-embed-event-card`)
- **External CSS compatibility**: Host site CSS can override embed styles via
  class selectors
- **CSS isolation**: Use CSS custom properties (variables) for themeable values
- **Responsive design**: Mobile-friendly layout that adapts to container width

### 5. API Integration

- **New embed HTML endpoint**: `/api/html/embed?userId=xyz` - Returns HTML
  fragment containing only the embed widget (no navbar, no branding, no footer)
  - Returns: Event list, event filters, event search
  - No MeetNearMe branding visible to end users
  - Used for generating embed codes in UI (future) and testing via CURL (now)
- **Reuse existing endpoint**: `/api/html/events` (already handles search/filter
  params)
- **Query parameters**: Support `q`, `start_time`, `end_time`, `radius`, `lat`,
  `lon`, `categories`
- **HTMX integration**: Use HTMX for dynamic event list updates (already
  implemented)
- **No authentication required**: Embed works without user login

### 6. Technical Implementation

#### Script Structure

```javascript
// Pseudo-code structure
(function () {
  // 1. Load dependencies (Alpine.js, HTMX if not present)
  // 2. Inject CSS (minimal, class-based)
  // 3. Create container element
  // 4. Render search bar
  // 5. Render filter UI (collapsible/modal)
  // 6. Render event list container
  // 7. Initialize Alpine.js stores and state
  // 8. Set up HTMX form submission
})();
```

#### Template Modifications

- **Create new component**: `templates/components/widget.templ` that composes:
  - Event search bar (reuse from `home.templ` with conditional logic)
  - Event filters (reuse from `navbar.templ` with conditional logic)
  - Event list rendering (reuse `EventsInner` component)
  - No navbar, footer, or branding elements
- **Add `isEmbed` parameter** to existing template functions:
  - `Home()` - conditionally hide navbar/footer/branding
  - `Navbar()` - conditionally hide branding, show only filters
- **Conditional rendering**: Use `isEmbed` flag to show/hide components in
  existing templates
- **Reuse existing components**: No need to extract/duplicate - just
  conditionally render

### 7. Cross-Origin Testing Infrastructure

- **Why it's critical**: Embed scripts load from a different origin than the
  host website, requiring:
  - CORS headers on script endpoint
  - CORS headers on API endpoints (`/api/html/events`)
  - Testing with real cross-origin scenarios (not just localhost)
- **Tunneling Options** (for creating publicly accessible HTTPS URL for local
  development):

  **Recommended: Cloudflare Tunnel (cloudflared)** - Most stable, free, no
  signup required

  - Install: `brew install cloudflare/cloudflare/cloudflared` (macOS) or
    download from
    [cloudflare.com](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/)
  - Run: `cloudflared tunnel --url http://localhost:8000`
  - Provides stable HTTPS URL that doesn't drop connections
  - No account required, works immediately

  **Alternative: ngrok** - More stable than localtunnel, free tier available

  - Install: `brew install ngrok/ngrok/ngrok` (macOS) or download from
    [ngrok.com](https://ngrok.com/download)
  - Sign up for free account at ngrok.com
  - Run: `ngrok http 8000`
  - More reliable than localtunnel, but requires account setup

  **Not Recommended: Localtunnel** - Unstable, connections drop frequently

  - Install: `npm install -g localtunnel`
  - Run: `lt --port 8000`
  - Known issues: Frequent connection drops, firewall problems, unstable service
  - Only use if other options aren't available

### 8. Initial Testing

- Create standalone `test_embed.html` file for testing
- Include script tag pointing to embed endpoint (via localtunnel URL)
- Test with various CSS frameworks (Bootstrap, Tailwind, custom CSS)
- Verify search, filters, and event rendering work correctly
- Test on external websites (GitHub Pages, personal sites) using localtunnel URL

### 9. Future Considerations (Out of Scope for Now)

- Embed customization UI (colors, layout options)
- User dashboard for generating embed codes
- Analytics/tracking for embed usage
- Multi-instance support (multiple embeds on same page)
- Custom event filtering per embed instance

## Implementation Phases

### Phase 1: Infrastructure Setup & CORS Configuration

**Goal**: Set up the development infrastructure and configure CORS so we can
test cross-origin loading.

- [ ] **Install and configure localtunnel**

  - Install: `npm install -g localtunnel`
  - Create tunnel script/command: `lt --port 8000` (Go app runs on port 8000,
    not 8080 which is used by Weaviate)
  - Document the localtunnel URL format for team reference
  - Verify tunnel works by accessing local server via tunnel URL

- [ ] **Configure CORS headers for embed script endpoint**

  - Add CORS middleware or headers to embed script route
  - Allow `Access-Control-Allow-Origin: *` (or specific domains for production)
  - Set `Access-Control-Allow-Methods: GET`
  - Ensure `Content-Type: application/javascript` or `text/javascript`
  - Test CORS headers with browser DevTools Network tab

- [ ] **Configure CORS headers for API endpoints**

  - Update `/api/html/events` endpoint to allow cross-origin requests
  - Add CORS headers: `Access-Control-Allow-Origin`,
    `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`
  - Test with browser DevTools Network tab to verify headers

### Phase 2: Hello World Embed & Basic Testing

**Goal**: Create a minimal working embed to verify the basic mechanism works
before adding complexity.

- [ ] **Create simple "Hello World" embed script**

  - Create minimal embed script endpoint (e.g., `/embed.js` or
    `/api/embed/script`)
  - Script injects a simple "Hello World" message into a target container
  - Include basic structure: script loads, finds container, injects HTML
  - Add CORS headers to script endpoint (from Phase 1)

- [ ] **Create test infrastructure**

  - Create `test_embed.html` file with container element (e.g.,
    `<div id="mnm-embed-container"></div>`)
  - Load embed script from localhost
  - Verify script loads correctly and "Hello World" appears
  - Test script loading from different origin (localtunnel URL)
  - Verify CORS headers are present and correct
  - Document any CORS-related issues found

- [ ] **Visual testing with Hello World embed**

  - Load embed script from localtunnel URL in test HTML
  - Verify "Hello World" message appears correctly in container
  - Test in multiple browsers (Chrome, Firefox, Safari)
  - Document visual appearance and any rendering issues

### Phase 3: CSS Loading & Cross-Origin Verification

**Goal**: Verify Tailwind CSS loads correctly in the embed and test full
cross-origin scenario.

- [ ] **Test Tailwind CSS loading in embed**

  - Add Tailwind CSS to the "Hello World" embed
    - Options: Load from CDN, inject compiled CSS, or load from static assets
    - Note: App uses DaisyUI plugin - may need to include DaisyUI styles or use
      CDN version
  - Use Tailwind utility classes in the test embed (e.g.,
    `bg-blue-500 text-white p-4 rounded-lg shadow-md`)
  - Verify Tailwind classes render correctly when embed is loaded
  - Test that external CSS doesn't conflict with embed styles
  - Document any CSS loading or scoping issues
  - Document final approach chosen for Tailwind inclusion in embed

- [ ] **Test cross-origin script loading**
  - Deploy test HTML to external location (GitHub Pages, personal site, or
    simple HTTP server)
  - Load embed script from localtunnel URL
  - Verify no CORS errors in browser console
  - Verify script executes correctly when loaded cross-origin
  - Verify Tailwind CSS loads correctly cross-origin
  - Verify styled "Hello World" appears correctly on external site

### Phase 4: Core Extraction & HTML Endpoint

- [ ] **Create embed widget component**

  - Create `templates/components/widget.templ` that composes:
    - Event search bar (reuse from `home.templ`)
    - Event filters (reuse from `navbar.templ`)
    - Event list rendering (reuse `EventsInner` component)
  - Widget component should accept same parameters as needed (events, userInfo,
    etc.)
  - No navbar, footer, or MeetNearMe branding in widget

- [ ] **Create `/api/html/embed` endpoint**

  - Handler in `page_handlers.go` or `partial_handlers.go`
  - Accepts `userId` query parameter: `/api/html/embed?userId=xyz`
  - Returns HTML fragment using `components.Widget()` template
  - Include CORS headers (from Phase 1)
  - Test via CURL: `curl "http://localhost:8000/api/html/embed?userId=xyz"`

- [ ] **Add `isEmbed` parameter to existing templates**

  - Add `isEmbed` boolean to `Home()` and `Navbar()` template functions
  - Conditionally show/hide components based on `isEmbed` flag:
    - In `Home()`: hide navbar, footer, branding when `isEmbed=true`
    - In `Navbar()`: hide branding, show only filter UI when `isEmbed=true`
  - Keep existing components - just conditionally render them

- [ ] **Extract and test components**
  - Verify search bar works in isolated template
  - Verify filter UI works in isolated template
  - Verify event list rendering works correctly

### Phase 5: Script Development

- [ ] **Create embed script endpoint** (separate from HTML endpoint)

  - Endpoint that returns JavaScript (e.g., `/embed.js` or `/api/embed/script`)
  - Script loads HTML from `/api/html/embed?userId=xyz` and injects into DOM
  - Include CORS headers for cross-origin script loading

- [ ] **Build self-contained script**
  - Script that injects widget HTML into target container
  - Implement dependency loading (Alpine.js, HTMX if not present)
  - Add CSS injection with embed-prefixed classes
  - Set up Alpine.js state management
  - Configure HTMX for event updates (verify CORS works)

## GetEmbedScript Function Specification

### Overview

The `GetEmbedScript` function in
`functions/gateway/handlers/partial_handlers.go` is responsible for generating
and serving the JavaScript embed script. This script is loaded by external
websites via a `<script>` tag and handles all the logic for loading
dependencies, fetching widget HTML, and initializing the widget.

### Function Signature

```go
func GetEmbedScript(w http.ResponseWriter, r *http.Request) http.HandlerFunc
```

### HTTP Response Requirements

1. **Content-Type**: `application/javascript` or `text/javascript`
2. **CORS Headers**: Must set CORS headers using
   `transport.SetCORSHeaders(w, r)` to allow cross-origin loading
3. **Response Body**: JavaScript code as a string (not a template - raw
   JavaScript)

   **Why raw JavaScript?** When a browser loads a script tag like
   `<script src="/api/embed.js"></script>`, it expects to receive JavaScript
   code that it can execute. The Go function generates this JavaScript code as a
   string and writes it directly to the HTTP response. This is different from
   HTML templates (like `templ`) - we're not rendering HTML, we're generating
   JavaScript code that will run in the browser. The JavaScript string is built
   in Go (using string concatenation or `fmt.Sprintf`) and sent as-is to the
   browser, which then executes it.

### JavaScript Script Responsibilities

The generated JavaScript must be an immediately-invoked function expression
(IIFE) that:

#### 1. Container Setup

- Find or create container element (default: `#mnm-embed-container` or
  `data-mnm-container` attribute)
- If container doesn't exist, create it and append to `document.body`
- Log container creation for debugging

#### 2. User ID Detection

- Get `userId` from:
  1. `data-user-id` attribute on the script tag (primary method)
  2. Query parameter in the script URL (fallback)
  3. If neither exists, show error message in container

#### 3. Base URL Detection

- Determine base URL for API calls:
  1. `STATIC_BASE_URL` environment variable (if set)
  2. `http://localhost:8001` for local development
  3. Current script's origin (fallback)
- Construct URLs for CSS and API endpoints using this base URL

#### 4. Dependency Loading

- Check if dependencies are already loaded on the host page:
  - **Alpine.js**: Check for `window.Alpine`
  - **HTMX**: Check for `window.htmx`
  - **Tailwind CSS**: Check for Tailwind CDN script or compiled CSS
  - **Main CSS**: Check for `styles.css` or `styles.*.css` (with hash)
  - **Google Fonts**: Check if fonts are loaded
- Load missing dependencies:
  - Alpine.js: Load from CDN if not present
  - HTMX: Load from CDN if not present
  - Tailwind CSS: Load from CDN (JavaScript-based) if not present
  - Main CSS: Load from static server (`/static/assets/styles.css` or hashed
    version)
  - Google Fonts: Load from Google Fonts CDN
- Log each dependency's status (loading, loaded, failed) to console for
  debugging
- Wait for all dependencies to load before proceeding

#### 5. Widget HTML Fetching

- Fetch HTML from `/api/html/embed?userId={userId}` endpoint
- Use `fetch()` API with proper headers:
  - `Accept: text/html`
  - `credentials: 'omit'` (no cookies needed)
- Handle fetch errors gracefully with user-friendly error message in container

#### 6. HTML Parsing & Script Extraction

- **Critical Issue**: `innerHTML` does NOT execute `<script>` tags, so scripts
  must be extracted and executed manually
- Parse fetched HTML using `DOMParser` to create a document fragment
- Extract all `<script>` tags from the parsed HTML:
  - Iterate through all script elements in the parsed document
  - For each script, create a unique marker comment in the HTML
  - Marker format: `<!-- MNM_SCRIPT_MARKER:{index}:{scriptId} -->` where:
    - `{index}` is the sequential index of the script (0, 1, 2, etc.)
    - `{scriptId}` is the script's `id` attribute if present, or
      `inline-{index}` for inline scripts
  - Store script metadata (attributes, content, original position) in an array
- Replace each `<script>` tag in the HTML with its corresponding marker comment
- This preserves the exact DOM structure and script execution order
- **Important**: Scripts must maintain their original position in the DOM
  hierarchy for proper execution context

#### 7. HTML Injection & Script Execution

- Inject the modified HTML (with script markers) into the container using
  `innerHTML`
- **Critical**: HTML must be injected BEFORE scripts execute so elements like
  `#alpine-state` exist when stores try to access them
- After HTML injection, locate each script marker comment in the DOM:
  - Query for all marker comments: `container.querySelectorAll('*')` and filter
    for comment nodes, or use `TreeWalker` to find comment nodes
  - For each marker comment found:
    - Extract the script index and ID from the marker
    - Retrieve the corresponding script data from the stored array
    - Create a new `<script>` element
    - Copy all attributes from the original script (id, src, type, defer, async,
      etc.)
    - Copy script content (for inline scripts) or set `src` attribute (for
      external scripts)
- Insert each script element at the exact position of its marker comment:
  - Use `markerComment.parentNode.insertBefore(newScript, markerComment)`
  - Remove the marker comment after insertion
  - This ensures scripts execute in their original order and DOM context
- **Execution Order**: Scripts execute synchronously as they are inserted
  (inline scripts) or asynchronously when loaded (external scripts with `src`)
- **Error Handling**:
  - Wrap script insertion in try/catch blocks to catch execution errors
  - For inline scripts: Wrap script content execution in try/catch to catch
    runtime errors without blocking other scripts
  - For external scripts: Add `onerror` handlers to catch load failures
  - Log errors with script index/ID for debugging
  - Continue with remaining scripts even if one fails
- **Execution Verification**:
  - After inserting inline scripts, verify they executed by checking for
    expected side effects (e.g., functions defined, event listeners added,
    console logs appearing)
  - If a script should define functions or add event listeners, verify they
    exist after insertion
  - Add timeout-based verification for scripts that should execute immediately
  - Log verification results (success/fail) for debugging

#### 8. Alpine Store Registration

- **Critical Timing Issue**: Stores must be registered BEFORE Alpine processes
  HTML
- After scripts execute, trigger `alpine:init` event:

  ```javascript
  document.dispatchEvent(new CustomEvent('alpine:init', { bubbles: true }));
  ```

- Wait for stores to register (check `Alpine.store('urlState')`,
  `Alpine.store('filters')`, `Alpine.store('location')`)
- If stores not registered, retry `alpine:init` event
- Log store registration status for debugging

#### 9. Alpine Initialization

- Check if Alpine is already initialized on host page:
  - Check `window.Alpine._initialized` flag
  - Or check if `document.body` has `[x-data]` attributes
- If already initialized:
  - Use `Alpine.initTree(container)` to process only the new HTML
  - Prevents double initialization warnings
- If not initialized:
  - Call `Alpine.start()` to initialize Alpine
- Verify stores are accessible before initializing Alpine
- Log initialization method used

<!-- #### 10. Form Submission Handling

- Prevent default form submissions (sandbox protection for iframes):
  - Remove `action` attributes from all forms
  - Add event listeners in capture phase to prevent default submission
  - Manually trigger Alpine/HTMX handlers:
    - Search forms: Call `Alpine.store('urlState').setParam('q', value)`
    - HTMX forms: Call `htmx.trigger(form, 'submit')`
- Log form submission handling for debugging -->

#### 11. HTMX Initialization

- After Alpine is initialized, process HTMX elements:
  - Call `htmx.process(container)` to initialize HTMX on new HTML
  - This enables HTMX form submissions and dynamic updates

#### 12. Error Handling

- Catch and log all errors to console with "MeetNearMe Embed:" prefix
- Display user-friendly error messages in container if critical errors occur
- Don't throw uncaught errors that could break the host page

### Implementation Structure

The function should:

1. Set CORS headers
2. Set `Content-Type: application/javascript`
3. Generate JavaScript code as a Go template or string
4. Write JavaScript to response writer

### JavaScript Code Template Structure

```javascript
(function () {
  'use strict';

  // 1. Container setup
  // 2. User ID detection
  // 3. Base URL detection
  // 4. Dependency loading (with promises)
  // 5. Widget HTML fetching
  // 6. HTML parsing & script extraction (with comment markers)
  // 7. HTML injection & script execution (in original positions)
  // 8. Alpine store registration
  // 9. Alpine initialization
  // 10. Form submission handling
  // 11. HTMX initialization
  // 12. Error handling (try/catch around critical sections)
})();
```

### Key Challenges Addressed

1. **Script Execution**: `innerHTML` doesn't execute scripts, so we must
   manually extract and execute them
2. **Script Positioning**: Scripts must execute in their original DOM positions
   to maintain proper execution context and order
3. **Store Registration Timing**: Stores must exist before Alpine processes HTML
   expressions
4. **Alpine Double Initialization**: Host page may already have Alpine
   initialized
5. **Cross-Origin Loading**: All dependencies and API calls must work
   cross-origin
6. **Form Submission in Sandboxed Contexts**: Prevent browser form submission,
   use JavaScript handlers
7. **Dependency Detection**: Check if dependencies already exist before loading
   duplicates

### Testing Considerations

- Test with Alpine.js already loaded on host page
- Test with HTMX already loaded on host page
- Test with neither Alpine.js nor HTMX loaded
- Test with various CSS frameworks on host page
- Test cross-origin loading (via tunnel)
- Test error scenarios (missing userId, network failures, etc.)
- Verify console logs are helpful for debugging

### Phase 6: Testing & Refinement

- [ ] Test embed script via localtunnel on external websites
- [ ] Test with various host site CSS frameworks
- [ ] Verify responsive behavior
- [ ] Test search and filter functionality cross-origin
- [ ] Ensure no CSS conflicts
- [ ] Verify API calls work from embedded context

## Key Files to Modify

- `functions/gateway/main.go` - Add CORS middleware/headers for embed and API
  endpoints
- `functions/gateway/handlers/page_handlers.go` - Create `/api/html/embed`
  endpoint handler
- `functions/gateway/handlers/partial_handlers.go` - Add CORS headers to
  `/api/html/events` endpoint, potentially add `/api/html/embed` handler here
- `functions/gateway/templates/pages/home.templ` - Add `isEmbed` parameter and
  conditional logic to hide navbar/footer when embed
- `functions/gateway/templates/components/widget.templ` - **NEW FILE**: Create
  widget component that composes search, filters, and event list (no branding)
- `functions/gateway/templates/components/navbar.templ` - Add `isEmbed`
  parameter to conditionally hide branding, show only filters when embed

## Notes

- Widget should be responsive and work in containers of various widths
- Consider lazy-loading events on initial render
- Ensure accessibility (ARIA labels, keyboard navigation)
- Test cross-browser compatibility
- **CORS is critical**: The embed script and all API endpoints must have proper
  CORS headers to work on external websites
- **Localtunnel workflow**: During development, run `lt --port <port>` to get
  public URL, use that URL in test HTML files on external sites
- **Two endpoints**: `/api/html/embed` (HTML fragment) and embed script endpoint
  (JavaScript) are separate - HTML endpoint can be tested directly via CURL
- **No branding in embed**: The `/api/html/embed` endpoint must return HTML
  without any MeetNearMe branding, navbar, or footer
- Consider rate limiting for embed endpoints to prevent abuse
