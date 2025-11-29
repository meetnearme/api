# Embed Widget Issues - Debugging Checklist

## Critical Issues

### 1. CORS Error: Main CSS Files Blocked

**Status:** 🔴 Critical **Error:**
`Access to CSS stylesheet at 'http://localhost:8000/assets/styles.82a6336e.css' from origin 'https://app.beehiiv.com' has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present on the requested resource.`

**Problem:**

- The embed script tries to load CSS from `localhost:8000` when embedded on
  `https://app.beehiiv.com`
- The Go server is not sending CORS headers for static asset requests
- Both hashed (`styles.82a6336e.css`) and fallback (`styles.css`) versions fail

**Impact:**

- Main compiled CSS is not loading
- Widget has "limited styling" as warned in console
- Missing all custom styles, DaisyUI components, and theme variables

**Solution Needed:**

- Add CORS headers to static file serving in Go application
- Or serve CSS from a CDN/static host that allows cross-origin
- Or inline critical CSS in the widget HTML response

---

### 2. Missing DaisyUI

**Status:** 🔴 Critical **Problem:**

- DaisyUI component library is not being loaded
- DaisyUI provides the drawer, modal, and other component styles
- Without it, components like the filter drawer won't display correctly

**Current State:**

- Only Tailwind CDN is loaded (runtime processor)
- DaisyUI styles are likely in the main CSS file that's blocked by CORS

**Solution Needed:**

- Load DaisyUI CSS separately, OR
- Fix CORS issue so main CSS (which includes DaisyUI) can load
- May need to check if DaisyUI is in the compiled CSS or needs separate loading

---

### 3. Filter Drawer Not Visible

**Status:** 🔴 Critical **Problem:**

- Checkbox mechanism works (can be checked/unchecked)
- Drawer does not fly in/out visually
- Checkbox should be invisible, drawer should animate

**Root Causes:**

1. Missing CSS (DaisyUI drawer styles blocked by CORS)
2. Missing Alpine.js Focus plugin (required for `x-trap` directive)
3. Drawer component may need specific classes that aren't loading

**Console Evidence:**

- `Alpine Warning: You can't use [x-trap] without first installing the "Focus" plugin`
- Drawer uses `x-trap="modalIsOpen"` and `x-trap="openedWithKeyboard"`

**Solution Needed:**

1. Load Alpine.js Focus plugin
2. Fix CSS loading (CORS issue)
3. Verify drawer HTML structure and classes are correct

---

### 4. Alpine.js Double Initialization

**Status:** 🟡 Warning **Error:**
`Alpine Warning: Alpine has already been initialized on this page. Calling Alpine.start() more than once can cause problems.`

**Problem:**

- Embed script calls `Alpine.start()` but Alpine is already initialized on the
  host page
- This can cause conflicts with stores and event handlers

**Solution Needed:**

- Check if Alpine is already initialized before calling `start()`
- Only initialize Alpine stores/functions if not already present
- Use `Alpine.plugin()` for plugins instead of re-initializing

---

### 5. Missing Alpine.js Focus Plugin

**Status:** ✅ Fixed

**Previous Error:**
`Alpine Warning: You can't use [x-trap] without first installing the "Focus" plugin`

**Problem (Resolved):**

- The drawer and modal components use `x-trap` directive
- Focus plugin is loaded on regular site (`@alpinejs/focus@3.x.x`) but was not
  in embed
- Without it, focus trapping (accessibility feature) doesn't work

**Solution Implemented:**

- Added Focus plugin detection to dependency checking
- Added Focus plugin loading logic to embed script
- Ensures Focus plugin loads BEFORE Alpine.js when both need to be loaded
- If Alpine is already loaded on host page, Focus plugin will auto-register when
  loaded
- Loads from:
  `https://cdn.jsdelivr.net/npm/@alpinejs/focus@3.x.x/dist/cdn.min.js`

---

### 6. Location Dropdown Not Working

**Status:** 🔴 Critical **Problem:**

**Symptoms:**

1. **Initial state is blank/empty**: The location dropdown shows empty/blank
   instead of displaying the selected location
2. **Dropdown doesn't open**: When clicking the location dropdown button,
   nothing happens - no console errors, dropdown doesn't open at all

**Root Cause:**

- **Initial state issue**: `selectedOption` in `getLocationSearchState()` is
  initialized by accessing `Alpine.store('location').selected.label`, which can
  fail if the store isn't ready or doesn't have the expected structure
- **Click handler issue**: The button has
  `x-on:click="!isLoading && (isOpen = ! isOpen)"` but child elements cannot
  access parent scope properties (`isLoading`, `isOpen`) due to Alpine scope
  inheritance problems
- This is related to the broader issue where `Alpine.initTree()` doesn't
  properly set up scope inheritance for child elements to access parent
  component data

**Console Evidence:**

- Sometimes errors like cdn.min.js:5 Uncaught ReferenceError: isLoading is not
  defined
- `cdn.min.js:5 Uncaught TypeError: Cannot read properties of undefined (reading '_x_refs')` -
  Alpine internal error when processing elements
- Diagnostic logs show parent element has data stack, but child elements don't
  inherit scope
- `isLoading is not defined` errors when Alpine tries to evaluate expressions on
  child elements

**Current State:**

- Parent element (`x-data="getLocationSearchState()"`) has component data with
  `isLoading` and `isOpen` properties
- Child elements (button, dropdown content) cannot access these properties
- Click handler expression fails because `isLoading` and `isOpen` are undefined
  in child scope

**Solution Needed:**

- Fix Alpine scope inheritance so child elements can access parent component
  data
- May require different initialization approach than `Alpine.initTree()`
- Consider using Alpine's internal scope resolution mechanisms
- Or restructure the component to avoid scope inheritance issues
- The `_x_refs` error suggests Alpine is processing elements before they're
  fully initialized - may need to ensure proper initialization order

**Related Errors:**

- `Cannot read properties of undefined (reading '_x_refs')` - Alpine internal
  error indicating elements aren't fully initialized when processed
- This is likely a symptom of the same scope inheritance/timing issue

---

### 7. Time Filter Buttons (TODAY, TOMORROW, etc.) Not Working

**Status:** 🔴 Critical **Problem:**

**Symptoms:**

- Clicking "TODAY", "TOMORROW", "THIS WEEK", "THIS MONTH" buttons does nothing
- Buttons should trigger a form submission to reload events filtered by the
  selected time period
- No console errors, buttons appear clickable but don't update the event list

**Root Cause:**

- Buttons use `@click="$store.urlState.setParam('start_time', 'today')"` to
  update the URL state
- The `setParam()` method should:
  1. Update the Alpine store
  2. Update the URL with `window.history.pushState()`
  3. Find the form `#event-search-form` and call `form.requestSubmit()`
  4. HTMX should then fetch new events and update `#events-container-inner`
- **Note**: Unlike the location dropdown, these buttons use `$store` (Alpine
  magic property) directly, not component scope - this is a **different issue**
- Likely issues (different from location dropdown):
  1. **Alpine not processing `@click` directive**: Button might not be in an
     Alpine context when processed
  2. **`$store` magic property not available**: Alpine might not have
     initialized the magic properties on the button
  3. **Form not found**: `document.getElementById('event-search-form')` might
     not find the form in embed context
  4. **HTMX not processing**: Form submission might not trigger HTMX request
     properly
  5. **Button not processed by Alpine.initTree()**: The button might not be
     getting processed when `Alpine.initTree()` is called

**Console Evidence:**

- No errors when clicking buttons (handler fails silently)
- Stores are verified as registered in console logs
- This is a **different issue** from the location dropdown scope inheritance
  problem

**Current State:**

- Buttons are visible and clickable
- **ROOT CAUSE IDENTIFIED**: Alpine's `@click` handlers are attached
  (`_x_attributeCleanups` exists), but they're not executing because `$store`
  magic property cannot be resolved in the button's context
- Manual click listeners work perfectly (calling
  `window.Alpine.store('urlState').setParam(...)` directly)
- This confirms: **Alpine scope inheritance is broken** - child elements cannot
  access parent's Alpine context, including magic properties like `$store`

**What We Learned:**

1. ✅ Stores are registered and accessible via `window.Alpine.store('urlState')`
2. ✅ `setParam()` function works correctly when called manually
3. ✅ Buttons have Alpine listeners attached (`_x_attributeCleanups: true`)
4. ❌ Buttons don't have Alpine data stack (`_x_dataStack: false`)
5. ❌ Parent has `x-data="getHomeState()"` but child buttons can't inherit scope
6. ❌ Alpine's `@click` handlers fail silently because `$store` resolves to
   `undefined` in button context

**Solution Needed:**

- **Primary Fix**: Ensure Alpine properly inherits parent scope for child
  elements when using `Alpine.initTree()`
- This is the same root cause as Issue #6 (Location Dropdown) - **Alpine scope
  inheritance is broken in dynamically injected HTML**
- Options:
  1. Fix Alpine scope inheritance (may require Alpine plugin or different
     initialization approach)
  2. Use manual event listeners as workaround (current temporary solution)
  3. Ensure buttons are explicitly within an Alpine context that has access to
     stores
  4. Consider using `Alpine.data()` components instead of `x-data` functions for
     better scope management

---

### 8. Search Form Submit Not Working in Embed

**Status:** 🔴 Critical **Problem:**

**Symptoms:**

- Clicking the search button (or pressing Enter) in the search form doesn't set
  the `q` parameter
- The form has
  `@submit.prevent="$store.urlState.setParam('q', document.getElementById('search-input').value)"`
- Works perfectly on the actual website, but fails silently in the embed
- No console errors, the form just doesn't submit/update the query parameter

**Root Cause:**

This is the same Alpine scope inheritance issue as Issues #6 and #7. The form's
`@submit.prevent` handler uses `$store.urlState.setParam()`, but `$store` cannot
be resolved in the form's context because:

1. The form is a child element within the widget
2. Alpine scope inheritance is broken when using `Alpine.initTree()` on
   dynamically injected HTML
3. `$store` magic property resolves to `undefined` in the form's context
4. The handler fails silently, so the form doesn't submit

**Console Evidence:**

- No errors when clicking search button
- Form submission handler doesn't execute
- `$store` is undefined in the form's Alpine context

**Current State:**

- Search input field is visible and typeable
- Search button is clickable
- But `@submit.prevent` handler doesn't execute because `$store` is undefined
- This prevents the `q` parameter from being set and the form from submitting

**Solution Needed:**

- **Same fix as Issues #6 and #7**: Fix Alpine scope inheritance for dynamically
  injected HTML
- **Temporary workaround**: Add a manual event listener to the search form
  (similar to what we did for time filter buttons)
- The manual listener would:
  1. Listen for form submit events
  2. Get the search input value
  3. Call `window.Alpine.store('urlState').setParam('q', value)` directly
  4. Prevent default form submission

**Related Issues:**

- Issue #6: Location Dropdown Not Working (same root cause)
- Issue #7: Time Filter Buttons Not Working (same root cause)
- All three issues stem from Alpine scope inheritance problems in embed context

---

### 9. Event Filtering Not Working (Query Params Update But Events Don't Filter)

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

### 10. Theme CSS Variables Missing

**Status:** ✅ Fixed

- Theme CSS variables (custom colors, etc.) are generated dynamically in
  `Layout` template
- Widget doesn't use `Layout`, so theme variables aren't included
- These are in a `<style>` tag generated by `themeStyleTag` function

**Solution Needed:**

- Include theme CSS variables in widget HTML response
- Or generate them in `GetEmbedHtml` handler and inject into widget HTML

---

### 11. Mixed Content Warning

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

1. **Fix CORS for CSS files** (Blocks all styling) ✅ Fixed
2. **Fix event filtering** (Query params update but events don't filter -
   CRITICAL) ✅ Fixed (sendParmsFromQs updated)
3. **Fix Alpine scope inheritance** (Location dropdown, time filter buttons,
   search form not working - CRITICAL)
4. **Verify DaisyUI loading** (Required for drawer components) ✅ Fixed
5. **Fix Alpine double initialization** (Prevents conflicts)
6. **Add theme CSS variables** (Completes styling) ✅ Fixed
7. **Handle mixed content** (Production concern)

---

## Notes

- The embed script correctly detects missing dependencies and attempts to load
  them
- All JavaScript dependencies (Alpine, HTMX) load successfully
- The core issue is CSS loading being blocked by CORS
- Drawer functionality depends on both CSS and Alpine Focus plugin
- Location dropdown issues are related to Alpine scope inheritance problems when
  using `Alpine.initTree()` on dynamically injected HTML
