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

**Status:** ✅ Fixed

### 10. CORS Error: Location API Endpoints Failing in Embed

**Status:** ✅ Fixed

### 11. Event Filtering Not Working (Query Params Update But Events Don't Filter)

**Status:** ✅ Fixed

## Medium Priority Issues

### 12. Theme CSS Variables Missing

**Status:** ✅ Fixed

### 13. Mixed Content Warning

**Status:** ✅ Fixed ??? (Haven't seen it since but maybe on local tunnel)

`Mixed Content: The page at 'about:srcdoc' was loaded over HTTPS, but requested an insecure element`

**Problem:**

- Embed tries to load resources over HTTP from `localhost:8000`
- Browser auto-upgrades to HTTPS, but this may cause issues

**Solution Needed:**

- Use HTTPS for local development (via localtunnel or similar)
- Or ensure production uses HTTPS URLs

## Non-Critical Issues

## Action Plan Priority

1. **Fix CORS for GET requests** (Preflight works but actual request fails) -
   **Status:** ✅ Fixed
2. **Fix CORS for CSS files** (Blocks all styling) ✅ Fixed
3. **Fix event filtering** (Query params update but events don't filter -
   CRITICAL) ✅ Fixed (sendParmsFromQs updated) ✅ Fixed
4. **Fix Alpine scope inheritance** (Location dropdown, time filter buttons,
   search form not working - CRITICAL) ✅ Fixed
5. **Verify DaisyUI loading** (Required for drawer components) ✅ Fixed
6. **Fix Alpine double initialization** (Prevents conflicts) ✅ Fixed
7. **Add theme CSS variables** (Completes styling) ✅ Fixed
8. **Handle mixed content** (Production concern)
