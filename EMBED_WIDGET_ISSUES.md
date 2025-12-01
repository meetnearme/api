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

### 13. CORS Error: Tailwind CSS CDN Blocked

**Status:** ✅ Fixed

### 14. Failed Resource Load Error

**Status:** ⚠️ Non-Issue (Local dev only)

The hashed CSS file doesn't exist locally but when using local tunnels this
error shows because it tries to fetch the hashed CSS. This should be a non-issue
in production.

### 15. Mixed Content Warning (Production - localhost:8001)

**Status:** ⚠️ Non-Issue (Local Development Only)

**Error Message:**

```text
Mixed Content: The page at 'https://use.meetnear.me/embed-code-test/?preview_id=...' was loaded over HTTPS, but requested an insecure element 'http://localhost:8001/static/assets/img/cat none [XX].jpeg'. This request was automatically upgraded to HTTPS.
```

**Why This Is Not a Problem:**

- Production environment variables are set correctly via deployment pipelines
- Local development with tunnels is expected to have this limitation
- The warnings don't affect functionality - they're just browser security
  warnings
- Images will load correctly in actual production deployments
