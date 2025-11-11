/**
 * Meet Near Me Embed Script
 * Embeds events from Meet Near Me into external websites
 */
(function () {
  'use strict';

  // Configuration
  const config = {
    htmxVersion: '1.9.12',
    alpineVersion: '3.x.x',
    htmxCDN: 'https://unpkg.com/htmx.org@',
    alpineCDN: 'https://cdn.jsdelivr.net/npm/alpinejs@',
    htmxJsonExtCDN: 'https://unpkg.com/htmx.org@',
    defaultSearchStyle: 'inline',
  };

  /**
   * Check if a script is already loaded
   */
  function isScriptLoaded(url) {
    const scripts = document.getElementsByTagName('script');
    for (let i = 0; i < scripts.length; i++) {
      if (scripts[i].src && scripts[i].src.includes(url)) {
        return true;
      }
    }
    return false;
  }

  /**
   * Check if a library is available globally
   */
  function isLibraryLoaded(name) {
    if (name === 'htmx') {
      return typeof window.htmx !== 'undefined';
    }
    if (name === 'alpine') {
      return typeof window.Alpine !== 'undefined';
    }
    return false;
  }

  /**
   * Load a script dynamically
   */
  function loadScript(src, onLoad) {
    if (
      isScriptLoaded(src) ||
      (src.includes('htmx') && isLibraryLoaded('htmx')) ||
      (src.includes('alpine') && isLibraryLoaded('alpine'))
    ) {
      if (onLoad) onLoad();
      return Promise.resolve();
    }

    return new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = src;
      script.async = true;
      script.onload = () => {
        if (onLoad) onLoad();
        resolve();
      };
      script.onerror = () => reject(new Error(`Failed to load script: ${src}`));
      document.head.appendChild(script);
    });
  }

  /**
   * Load CSS stylesheet
   */
  function loadCSS(href) {
    if (document.querySelector(`link[href="${href}"]`)) {
      return Promise.resolve();
    }

    return new Promise((resolve, reject) => {
      const link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = href;
      link.onload = () => resolve();
      link.onerror = () => reject(new Error(`Failed to load CSS: ${href}`));
      document.head.appendChild(link);
    });
  }

  /**
   * Detect API base URL from script src or use default
   */
  function detectApiBaseUrl() {
    const scripts = document.getElementsByTagName('script');
    for (let i = 0; i < scripts.length; i++) {
      const src = scripts[i].src;
      if (src && src.includes('mnm-embed.js')) {
        try {
          const url = new URL(src);
          return `${url.protocol}//${url.host}`;
        } catch (e) {
          // Invalid URL, continue
        }
      }
    }
    return 'https://meetnear.me';
  }

  /**
   * Initialize the embed
   */
  function init(options) {
    const opts = {
      ownerId: options.ownerId,
      target: options.target || document.querySelector('[data-mnm-embed]'),
      searchStyle: options.searchStyle || config.defaultSearchStyle,
      apiBaseUrl: options.apiBaseUrl || detectApiBaseUrl(),
    };

    if (!opts.ownerId) {
      console.error('Meet Near Me Embed: ownerId is required');
      return;
    }

    if (!opts.target) {
      console.error(
        'Meet Near Me Embed: Target element not found. Add data-mnm-embed attribute to container element.',
      );
      return;
    }

    // Set up loading state
    opts.target.innerHTML =
      '<div style="text-align: center; padding: 2rem;"><div class="loading loading-spinner loading-lg"></div><p>Loading events...</p></div>';

    // Load dependencies
    Promise.all([
      loadScript(`${config.htmxCDN}${config.htmxVersion}`),
      loadScript(`${config.htmxCDN}${config.htmxVersion}/dist/ext/json-enc.js`),
      loadScript(`${config.alpineCDN}${config.alpineVersion}/dist/cdn.min.js`),
    ])
      .then(() => {
        // Load Tailwind CSS (optional - only if not already loaded)
        const tailwindLoaded =
          document.querySelector('link[href*="tailwindcss"]') ||
          document.querySelector('style[data-tailwind]');

        if (!tailwindLoaded) {
          // Note: We're not loading Tailwind here as it's large
          // Users should include it themselves or we'll use inline styles
          console.warn(
            'Meet Near Me Embed: Tailwind CSS not detected. Some styles may not work correctly. Please include Tailwind CSS in your page.',
          );
        }

        // Build embed URL
        const embedUrl = new URL(`${opts.apiBaseUrl}/api/embed/events`);
        embedUrl.searchParams.set('ownerId', opts.ownerId);
        embedUrl.searchParams.set('searchStyle', opts.searchStyle);

        // Preserve existing query params with mnm_ prefix
        const currentParams = new URLSearchParams(window.location.search);
        const paramMap = {
          'mnm_q': 'mnm_q',
          'mnm_radius': 'mnm_radius',
          'mnm_lat': 'mnm_lat',
          'mnm_lon': 'mnm_lon',
          'mnm_start_time': 'mnm_start_time',
          'mnm_end_time': 'mnm_end_time',
          'mnm_location': 'mnm_location',
          'mnm_categories': 'mnm_categories',
        };
        
        // Also check for non-prefixed params for backward compatibility
        const legacyParams = {
          'q': 'mnm_q',
          'radius': 'mnm_radius',
          'lat': 'mnm_lat',
          'lon': 'mnm_lon',
          'start_time': 'mnm_start_time',
          'end_time': 'mnm_end_time',
          'location': 'mnm_location',
          'categories': 'mnm_categories',
        };
        
        // First check for mnm_ prefixed params
        Object.keys(paramMap).forEach((param) => {
          if (currentParams.has(param)) {
            embedUrl.searchParams.set(param, currentParams.get(param));
          }
        });
        
        // Then check for legacy params (only if mnm_ version doesn't exist)
        Object.keys(legacyParams).forEach((legacyParam) => {
          const mnmParam = legacyParams[legacyParam];
          if (!currentParams.has(mnmParam) && currentParams.has(legacyParam)) {
            embedUrl.searchParams.set(mnmParam, currentParams.get(legacyParam));
          }
        });

        // Fetch and inject embed content
        fetch(embedUrl.toString())
          .then((response) => {
            if (!response.ok) {
              throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.text();
          })
          .then((html) => {
            opts.target.innerHTML = html;

            // Initialize Alpine if not already done
            if (window.Alpine && !window.Alpine.isStarted) {
              window.Alpine.start();
            }
          })
          .catch((error) => {
            console.error(
              'Meet Near Me Embed: Failed to load embed content',
              error,
            );
            opts.target.innerHTML = `
						<div style="padding: 2rem; text-align: center; border: 1px solid #e5e7eb; border-radius: 0.5rem;">
							<p style="color: #ef4444;">Failed to load events. Please check your configuration.</p>
							<p style="color: #6b7280; font-size: 0.875rem; margin-top: 0.5rem;">Error: ${error.message}</p>
						</div>
					`;
          });
      })
      .catch((error) => {
        console.error('Meet Near Me Embed: Failed to load dependencies', error);
        opts.target.innerHTML = `
				<div style="padding: 2rem; text-align: center; border: 1px solid #e5e7eb; border-radius: 0.5rem;">
					<p style="color: #ef4444;">Failed to load required libraries.</p>
					<p style="color: #6b7280; font-size: 0.875rem; margin-top: 0.5rem;">Error: ${error.message}</p>
				</div>
			`;
      });
  }

  // Export API
  window.MeetNearMeEmbed = {
    init: init,
    version: '1.0.0',
  };

  // Auto-initialize if data attributes are present
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      const autoTarget = document.querySelector('[data-mnm-embed]');
      if (autoTarget && autoTarget.dataset.mnmOwnerId) {
        init({
          ownerId: autoTarget.dataset.mnmOwnerId,
          target: autoTarget,
          searchStyle:
            autoTarget.dataset.mnmSearchStyle || config.defaultSearchStyle,
          apiBaseUrl: autoTarget.dataset.mnmApiBaseUrl,
        });
      }
    });
  } else {
    const autoTarget = document.querySelector('[data-mnm-embed]');
    if (autoTarget && autoTarget.dataset.mnmOwnerId) {
      init({
        ownerId: autoTarget.dataset.mnmOwnerId,
        target: autoTarget,
        searchStyle:
          autoTarget.dataset.mnmSearchStyle || config.defaultSearchStyle,
        apiBaseUrl: autoTarget.dataset.mnmApiBaseUrl,
      });
    }
  }
})();
