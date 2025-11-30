# Embed Widget Issues - Debugging Checklist

## Critical Issues

### 1. CORS Error: Main CSS Files Blocked

**Status:** ✅ Fixed

### 2. Missing DaisyUI

**Status:** ✅ Fixed

### 3. Filter Drawer Not Visible

**Status:** ✅ Fixed

### 4. Alpine.js Double Initialization

**Status:** ✅ Fixed

### 5. Missing Alpine.js Focus Plugin

**Status:** ✅ Fixed

### 6. Location Dropdown Not Working

**Status:** ✅ Fixed

### 7. Time Filter Buttons (TODAY, TOMORROW, etc.) Not Working

**Status:** ✅ Fixed

### 8. Search Form Submit Not Working in Embed

**Status:** ✅ Fixed

### 9. CORS Error: Preflight Works But Actual GET Request Fails

**Status:** 🔴 Critical **Problem:**

**Symptoms:**

- OPTIONS preflight requests succeed (return 200 OK with CORS headers)
- Actual GET requests to `/api/html/events` fail with CORS error
- Browser Network tab shows:
  - ✅ OPTIONS request: Status 200, Type: `preflight`
  - ❌ GET request: Status: `CORS error`, Type: `xhr`
- Both requests go to the same URL with the same query parameters

**Root Cause (Under Investigation):**

The preflight OPTIONS request is handled correctly and returns proper CORS
headers, but the actual GET request response is missing or has incorrect CORS
headers, causing the browser to reject it.

**What We've Tried:**

1. **Moved CORS headers to start of handler function**

   - Set CORS headers at the beginning of `GetEventsPartial` before any
     processing
   - Ensures headers are set even on early returns
   - **Result:** Still fails

2. **Added OPTIONS method support to route registration**

   - Modified `addRoute()` to include OPTIONS method for public routes
   - Ensures OPTIONS requests match the route
   - **Result:** Preflight works, but GET still fails

3. **Fixed CORS credentials conflict**

   - Removed `Access-Control-Allow-Credentials: true` when using
     `Access-Control-Allow-Origin: *`
   - Only set credentials when we have a specific origin
   - **Result:** Still fails

4. **Set CORS headers in SendHtmlRes**

   - Added CORS header setting inside `SendHtmlRes` handler function
   - Ensures headers are on the actual response writer that writes the response
   - **Result:** Still fails

5. **Set CORS headers in error handlers**

   - Added CORS headers to `SendHtmlErrorPartial` for error responses
   - **Result:** Still fails

6. **Set CORS headers before calling SendHtmlRes**
   - Call `SetCORSHeaders()` in `GetEventsPartial` before calling `SendHtmlRes`
   - Ensures headers are on the response writer before any processing
   - **Result:** Still fails

**Current CORS Header Implementation:**

```go
func SetCORSHeaders(w http.ResponseWriter, r *http.Request) {
    origin := r.Header.Get("Origin")
    if origin != "" {
        w.Header().Set("Access-Control-Allow-Origin", origin)
        w.Header().Set("Access-Control-Allow-Credentials", "true")
    } else {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        // Cannot use credentials with *
    }
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
}
```

**Possible Issues to Investigate:**

1. **Header order/timing**: Headers might need to be set in a specific order or
   before `WriteHeader()` is called
2. **Response writer wrapping**: The response writer might be wrapped by
   middleware, causing headers to be lost
3. **Header overwriting**: Something might be overwriting CORS headers after
   they're set
4. **Browser caching**: Browser might be caching a previous failed response
5. **HTMX-specific headers**: HTMX might be sending additional headers that need
   to be allowed
6. **Content-Type header conflict**: Setting `Content-Type: text/html` might
   conflict with CORS headers
7. **Multiple header setting**: Headers might be getting set multiple times,
   causing conflicts

**Next Steps to Debug:**

1. **Check actual response headers**: Use browser DevTools Network tab to
   inspect the actual GET response headers

   - Compare OPTIONS response headers vs GET response headers
   - Verify all CORS headers are present in GET response
   - Check for any conflicting headers
   - **UPDATE**: Browser shows "no data found" for GET request headers - this
     suggests the browser is blocking the response before it can be inspected,
     or the request is failing at a network level before headers are received

2. **Add logging**: Add server-side logging to verify CORS headers are being set

   - Log headers before and after `SendHtmlRes` call
   - Log headers in `SendHtmlRes` handler function
   - Verify headers are on the response writer

3. **Test with curl**: Use curl to make requests and inspect headers

   - `curl -H "Origin: https://example.com" -v http://localhost:8001/api/html/events`
   - Compare OPTIONS vs GET response headers

4. **Check middleware**: Verify no middleware is interfering with CORS headers

   - Check if middleware wraps the response writer
   - Check if middleware modifies headers

5. **Test simple endpoint**: Create a minimal test endpoint to isolate the issue

   - Simple handler that just sets CORS headers and returns HTML
   - Verify if the issue is specific to `GetEventsPartial` or general

6. **Check HTMX request headers**: Verify what headers HTMX is sending

   - Check `Access-Control-Request-Headers` in OPTIONS request
   - Ensure all requested headers are in `Access-Control-Allow-Headers`

7. **Investigate "no data found" issue**: Since browser shows "no data found"
   for GET response headers
   - This suggests the browser is blocking the response before headers are
     received
   - Could indicate the request is being rejected at the CORS preflight stage
     even though OPTIONS returns 200
   - Check if there's a mismatch between what OPTIONS promises and what GET
     delivers
   - Verify the actual HTTP response is being sent (not just headers)
   - Test with a simple curl request to see if server is actually responding:
     `curl -H "Origin: https://example.com" -v http://localhost:8001/api/html/events?q=test`
   - Check server logs to see if GET request is actually reaching the handler
   - Verify the response is not empty or malformed

**Related Code Locations:**

- `functions/gateway/handlers/partial_handlers.go`: `GetEventsPartial()`
  function
- `functions/gateway/transport/http.go`: `SetCORSHeaders()` and `SendHtmlRes()`
  functions
- `functions/gateway/main.go`: Route registration and OPTIONS method handling

---

### 10. Event Filtering Not Working (Query Params Update But Events Don't Filter)

**Status:** 🔴 Critical **Problem:**

**Symptoms:**

- Query parameters in URL update correctly (e.g., `?q=Dance&start_time=today`)
- But the events displayed don't match the filters
- Example: Search for "Dance" but see trivia events instead
- This affects all filters: search query, time filters, location filters

**Root Cause (IDENTIFIED):**

The form uses `hx-vals="js:{...sendParmsFromQs()}"` which tells HTMX to get
parameters from a JavaScript function `sendParmsFromQs()`. This function reads
from `window.location.search`:

```javascript
function sendParmsFromQs() {
  const urlParams = new URLSearchParams(window.location.search);
  return {
    start_time: urlParams.get('start_time'),
    q: urlParams.get('q'),
    // etc...
  };
}
```

**The Problem:**

1. When `setParam()` is called, it updates the URL using
   `window.history.pushState()`
2. `pushState()` updates the URL in the browser bar BUT does NOT update
   `window.location.search` (it's a read-only property that reflects the actual
   page location from when it loaded)
3. Then `form.requestSubmit()` is called
4. HTMX executes `sendParmsFromQs()` which reads from `window.location.search`
5. **`window.location.search` still has the OLD values** because `pushState()`
   doesn't update it!
6. So HTMX sends the OLD query parameters, not the new ones

**Why This Only Affects the Embed (Hypothesis):**

The same code issue exists on both the actual website and the embed, but it
might work on the actual website because:

1. **Hidden inputs might be working**: Even though `hx-vals` should take
   precedence, HTMX might fall back to form inputs if `hx-vals` evaluation fails
   or returns empty values. On the actual site, the hidden inputs with
   `:value="$store.urlState.*"` bindings might be properly reactive and
   providing the values.

2. **Timing differences**: On the actual site, there might be more time between
   `pushState()` and form submission, allowing something to sync.

3. **Different initialization**: The actual site might initialize Alpine/HTMX
   differently, making the hidden inputs more reliable.

In the embed context, the issue is more obvious because:

- The hidden inputs might not be reactive (Alpine scope inheritance issues)
- `hx-vals` is the primary mechanism, and it's reading stale values
- The embed context might have different timing or initialization order

**The Real Fix:**

Regardless of why it works on the actual site, the fix is the same: Update
`sendParmsFromQs()` to read from the Alpine store (Option 2) instead of
`window.location.search`. This makes it work reliably in both contexts and
doesn't depend on URL parsing or timing.

**Why Hidden Inputs Don't Help:**

The form has hidden inputs with `:value="$store.urlState.*"` bindings, but HTMX
ignores them because `hx-vals` takes precedence. HTMX uses `hx-vals` (which
calls `sendParmsFromQs()`) instead of reading the form inputs.

**The Fix:**

Update `sendParmsFromQs()` to read from the current URL (which `pushState()`
updated) instead of `window.location.search`. Use
`new URL(window.location.href).searchParams` or read directly from the Alpine
store.

**What We Know:**

- ✅ Query parameters update in URL (confirmed by user)
- ✅ Manual `setParam()` calls work (we can see URL change)
- ❌ Events don't actually filter based on parameters
- ❌ This affects all filter types (search, time, location)

**Debugging Needed:**

1. Check if HTMX request is being sent (Network tab)
2. Verify form parameters in HTMX request payload
3. Check HTMX response to see what events are returned
4. Verify HTMX target element exists and is being updated
5. Check if form submission is actually happening
6. Verify hidden input values are set before form submission

**Solution Needed:**

**Fix `sendParmsFromQs()` function** to read from the current URL (which
`pushState()` updated) instead of `window.location.search`:

**Option 1 (Recommended)**: Read from `window.location.href`:

```javascript
function sendParmsFromQs() {
  const url = new URL(window.location.href); // This includes pushState() changes
  const urlParams = url.searchParams;
  return {
    // ... same as before but using urlParams from window.location.href
  };
}
```

**Option 2 (Better)**: Read directly from Alpine store (most reliable):

```javascript
function sendParmsFromQs() {
  const urlState = window.Alpine.store('urlState');
  return {
    ...(urlState.start_time ? { start_time: urlState.start_time } : {}),
    ...(urlState.q ? { q: urlState.q } : {}),
    // etc - read directly from store
  };
}
```

**Why Option 2 is Better:**

- Store always has the current values (single source of truth)
- Doesn't depend on URL parsing
- Works even if URL hasn't updated yet
- More reliable in embed context

---

## Medium Priority Issues

### 11. Theme CSS Variables Missing

**Status:** ✅ Fixed

- Theme CSS variables (custom colors, etc.) are generated dynamically in
  `Layout` template
- Widget doesn't use `Layout`, so theme variables aren't included
- These are in a `<style>` tag generated by `themeStyleTag` function

**Solution Needed:**

- Include theme CSS variables in widget HTML response
- Or generate them in `GetEmbedHtml` handler and inject into widget HTML

---

### 12. Mixed Content Warning

**Status:** 🟢 Low (Auto-upgraded) **Warning:**
`Mixed Content: The page at 'about:srcdoc' was loaded over HTTPS, but requested an insecure element`

**Problem:**

- Embed tries to load resources over HTTP from `localhost:8000`
- Browser auto-upgrades to HTTPS, but this may cause issues

**Solution Needed:**

- Use HTTPS for local development (via localtunnel or similar)
- Or ensure production uses HTTPS URLs

---

## Non-Critical Issues

## Action Plan Priority

1. **Fix CORS for GET requests** (Preflight works but actual request fails -
   CRITICAL) 🔴
2. **Fix CORS for CSS files** (Blocks all styling) ✅ Fixed
3. **Fix event filtering** (Query params update but events don't filter -
   CRITICAL) ✅ Fixed (sendParmsFromQs updated)
4. **Fix Alpine scope inheritance** (Location dropdown, time filter buttons,
   search form not working - CRITICAL)
5. **Verify DaisyUI loading** (Required for drawer components) ✅ Fixed
6. **Fix Alpine double initialization** (Prevents conflicts)
7. **Add theme CSS variables** (Completes styling) ✅ Fixed
8. **Handle mixed content** (Production concern)

---

## Notes

- The embed script correctly detects missing dependencies and attempts to load
  them
- All JavaScript dependencies (Alpine, HTMX) load successfully
- The core issue is CSS loading being blocked by CORS
- Drawer functionality depends on both CSS and Alpine Focus plugin
- Location dropdown issues are related to Alpine scope inheritance problems when
  using `Alpine.initTree()` on dynamically injected HTML
