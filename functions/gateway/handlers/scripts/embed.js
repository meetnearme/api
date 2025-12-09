(function () {
  'use strict';

  const currentScript = document.currentScript;

  // Step #1: Container Setup
  let containerId = 'mnm-embed-container';
  let container = null;
  // Gets container with id "mnm-embed-container" if it already exists in the DOM
  container = document.getElementById(containerId);

  let customContainerId = null;
  if (currentScript) {
    customContainerId = currentScript.getAttribute('data-mnm-container');
  }

  if (customContainerId) {
    containerId = customContainerId;
    container = document.getElementById(containerId);
  }

  if (!container) {
    container = document.createElement('div');
    container.setAttribute('id', containerId);
    document.body.appendChild(container);
  }

  container.innerHTML =
    '<div class="p-3 bg-base-100 border-2 border-base-300 rounded-md"><h2>Loading Events ...</h2></div>';

  // Step #2: User ID Detection
  let userId = null;
  if (currentScript) {
    // First check data attribute
    userId = currentScript.getAttribute('data-user-id');

    // Then check script URL parameters
    if (!userId && currentScript.src) {
      try {
        // Handle both absolute and relative URLs
        const scriptUrl = currentScript.src.startsWith('http')
          ? currentScript.src
          : new URL(currentScript.src, window.location.href).href;
        const url = new URL(scriptUrl);
        userId = url.searchParams.get('userId');
      } catch (e) {
        console.warn('MeetNearMe Embed: Failed to parse script URL', e);
      }
    }
  }

  // Fallback: check current page URL for userId (works for both script tag and dynamic loading)
  if (!userId) {
    try {
      const url = new URL(window.location.href);
      userId = url.searchParams.get('userId');
    } catch (e) {
      console.warn('MeetNearMe Embed: Failed to parse page URL', e);
    }
  }

  if (!userId) {
    var errorMsg =
      '<div class="p-1 bg-red-100 border-2 border-red-500 rounded-md">MeetNearMe Embed Error: userId is required. Please add data-user-id="YOUR_USER_ID" to the script tag or include ?userId=YOUR_USER_ID in the script URL.</div>';
    container.innerHTML = errorMsg;
    return;
  }

  // Step #3: Base URL Detection
  const staticBaseUrlFromEnv = '%s';
  let staticBaseUrl;
  let baseUrl;

  if (staticBaseUrlFromEnv !== '') {
    staticBaseUrl = staticBaseUrlFromEnv;
  } else {
    staticBaseUrl = 'http://localhost:8001';
  }

  if (currentScript && currentScript.src) {
    const scriptUrl = new URL(currentScript.src);
    baseUrl = scriptUrl.origin;
  } else {
    baseUrl = window.location.origin;
  }

  // Step #4: Dependency Loading
  // Check if our CSS is already loaded by verifying it's from our domain and matches our paths
  const cssLinks = document.querySelectorAll('link[rel="stylesheet"]');
  let ourCssLoaded = false;
  for (let i = 0; i < cssLinks.length; i++) {
    const href = cssLinks[i].href || cssLinks[i].getAttribute('href') || '';
    if (
      href.includes(staticBaseUrl) &&
      (href.includes('styles.82a6336e.css') ||
        href.includes('/assets/styles.css') ||
        href.includes('/static/assets/styles.css'))
    ) {
      ourCssLoaded = true;
      break;
    }
  }

  const dependencies = {
    alpine: !!window.Alpine,
    htmx: !!window.htmx,
    tailwind: !!(
      document.querySelector('script[src*="tailwindcss.com"]') ||
      (window.tailwind && window.tailwind.config)
    ),
    mainCss: ourCssLoaded,
    fonts: !!document.querySelector('link[href*="fonts.googleapis.com"]'),
    focusPlugin: !!document.querySelector('script[src*="@alpinejs/focus"]'),
  };

  function loadScript(src) {
    return new Promise(function (resolve, reject) {
      var existing = document.querySelector('script[src="' + src + '"]');
      if (existing) {
        resolve();
        return;
      }
      const script = document.createElement('script');
      script.src = src;
      script.crossOrigin = 'anonymous';
      script.onload = resolve;
      script.onerror = function () {
        reject(new Error('Failed to load script: ' + src));
      };
      document.head.appendChild(script);
    });
  }

  function loadStylesheet(href) {
    return new Promise(function (resolve, reject) {
      const existing = document.querySelector('link[href="' + href + '"]');
      if (existing) {
        resolve();
        return;
      }
      const link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = href;
      link.crossOrigin = 'anonymous';
      link.onload = resolve;
      link.onerror = function () {
        reject(new Error('Failed to load stylesheet: ' + href));
      };
      document.head.appendChild(link);
    });
  }

  function createErrorPartial(message) {
    return (
      '<div role="alert" class="alert alert-error">' +
      '<svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-10 w-10" fill="none" viewBox="0 0 24 24">' +
      '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path>' +
      '</svg>' +
      '<div>' +
      message +
      '</div>' +
      '</div>'
    );
  }

  function insertErrorPartial(message) {
    const errorHtml = createErrorPartial(message);
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = errorHtml;
    const errorElement = tempDiv.firstChild;
    if (container.firstChild) {
      container.insertBefore(errorElement, container.firstChild);
    } else {
      container.appendChild(errorElement);
    }
  }

  let loadPromises = [];

  if (!dependencies.fonts) {
    loadPromises.push(
      loadStylesheet(
        'https://fonts.googleapis.com/css2?family=Outfit:wght@400&family=Ubuntu+Mono:ital,wght@0,400;0,700;1,400;1,700&family=Anton&family=Unbounded:wght@900&display=swap',
      ).catch(function (error) {
        console.log('MeetNearMe Embed: Failed to load Google Fonts', error);
      }),
    );
  }

  if (!dependencies.mainCss) {
    var cssBasePath = staticBaseUrl.endsWith('/static')
      ? '/assets/styles.css'
      : '/static/assets/styles.css';
    var cssHashedPath = staticBaseUrl.endsWith('/static')
      ? '/assets/styles.82a6336e.css'
      : '/static/assets/styles.82a6336e.css';
    loadPromises.push(
      loadStylesheet(staticBaseUrl + cssHashedPath).catch(function (error) {
        return loadStylesheet(staticBaseUrl + cssBasePath).catch(function (
          error,
        ) {
          console.log(
            'MeetNearMe Embed: Failed to load hashed CSS and fallback',
            error,
          );
        });
      }),
    );
  }

  if (!dependencies.focusPlugin) {
    loadPromises.push(
      loadScript(
        'https://cdn.jsdelivr.net/npm/@alpinejs/focus@3.x.x/dist/cdn.min.js',
      ).catch(function (error) {
        console.log(
          'MeetNearMe Embed: Failed to load Alpine focus plugin',
          error,
        );
      }),
    );
  }

  if (!dependencies.alpine) {
    loadPromises.push(
      loadScript(
        'https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js',
      ).catch(function (error) {
        console.log('MeetNearMe Embed: Failed to load Alpine.js', error);
      }),
    );
  }

  if (!dependencies.htmx) {
    loadPromises.push(
      loadScript('https://unpkg.com/htmx.org@1.9.10').catch(function (error) {
        console.log('MeetNearMe Embed: Failed to load HTMX', error);
      }),
    );
  }

  Promise.all(loadPromises).then(function () {
    if (!window.Alpine) {
      throw new Error('Alpine.js failed to load');
    }
    if (!window.htmx) {
      throw new Error('HTMX failed to load');
    }
    // Step #5: Widget HTML Fetching
    const embedUrl =
      baseUrl + '/api/html/embed?userId=' + encodeURIComponent(userId);
    fetch(embedUrl, {
      method: 'GET',
      headers: {
        Accept: 'text/html',
      },
      credentials: 'omit',
    })
      .then(function (response) {
        if (!response.ok) {
          throw new Error('Failed to load widget: HTTP ' + response.status);
        }
        return response.text();
      })
      .then(function (html) {
        // Step #6: HTML Parsing & Script Extraction
        let scriptsData = [];

        function findComments(element) {
          let markerComments = [];
          for (var j = 0; j < element.childNodes.length; j++) {
            var node = element.childNodes[j];
            if (
              node.nodeType === 8 &&
              node.nodeValue &&
              node.nodeValue.trim().indexOf('MNM_SCRIPT_MARKER:') === 0
            ) {
              markerComments.push(node);
            }
            if (node.nodeType === 1) {
              findComments(node);
            }
          }
          return markerComments;
        }

        function checkStores() {
          const requiredStores = ['urlState', 'filters', 'location'];
          const maxRetries = 20; // ~1 second total wait time (20 * 50ms)
          let retryCount = 0;

          function checkStoresInner() {
            if (!window.Alpine) {
              if (retryCount < maxRetries) {
                retryCount++;
                setTimeout(checkStoresInner, 50);
              }
              return;
            }

            const allRegistered = requiredStores.every(function (storeName) {
              return (
                window.Alpine.store &&
                typeof window.Alpine.store(storeName) !== 'undefined'
              );
            });

            if (allRegistered) {
              // Step #9: Alpine Initialization
              try {
                const isInitialized =
                  window.Alpine._initialized ||
                  document.body.querySelector('[x-data]') !== null;

                if (isInitialized) {
                  window.Alpine.initTree(container);
                } else {
                  window.Alpine.start();
                }
              } catch (error) {
                console.log(
                  'MeetNearMe Embed: Error initializing Alpine.js',
                  error,
                );
                insertErrorPartial(
                  'MeetNearMe Embed: Error initializing Alpine.js: ' +
                    error.message,
                );
              }

              // Step #11: HTMX Initialization
              try {
                if (window.htmx) {
                  window.htmx.process(container);
                }
              } catch (error) {
                console.log('MeetNearMe Embed: Error initializing HTMX', error);
                insertErrorPartial(
                  'MeetNearMe Embed: Error initializing HTMX: ' + error.message,
                );
              }

              return;
            }

            // Retry: dispatch event to trigger store registration, then check again
            if (retryCount < maxRetries) {
              retryCount++;
              document.dispatchEvent(
                new CustomEvent('alpine:init', { bubbles: true }),
              );
              setTimeout(checkStoresInner, 50);
            }
          }

          // Initial call: trigger Alpine initialization and start checking
          document.dispatchEvent(
            new CustomEvent('alpine:init', { bubbles: true }),
          );
          setTimeout(checkStoresInner, 50);
        }

        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');

        if (!doc || !doc.body) {
          throw new Error('Failed to parse HTML');
        }

        const scripts = doc.querySelectorAll('script');
        let htmlWithMarkers = html;

        for (var i = scripts.length - 1; i >= 0; i--) {
          const script = scripts[i];
          const scriptId = script.id || 'inline-' + i;
          const marker =
            '<!-- MNM_SCRIPT_MARKER:' + i + ':' + scriptId + ' -->';

          let scriptContent = script.textContent || '';
          // Convert relative URLs in fetch calls to absolute URLs
          if (scriptContent) {
            const templateLiteralPattern =
              'fetch(' + String.fromCharCode(96) + '/api/';
            const templateLiteralReplacement =
              'fetch(' + String.fromCharCode(96) + baseUrl + '/api/';
            scriptContent = scriptContent
              .split(templateLiteralPattern)
              .join(templateLiteralReplacement);
            scriptContent = scriptContent
              .split('fetch("/api/')
              .join('fetch("' + baseUrl + '/api/');
            scriptContent = scriptContent
              .split("fetch('/api/")
              .join("fetch('" + baseUrl + '/api/');
          }

          const scriptData = {
            index: i,
            id: scriptId,
            attributes: {},
            content: scriptContent,
            src: script.src || '',
          };

          for (let j = 0; j < script.attributes.length; j++) {
            const attr = script.attributes[j];
            scriptData.attributes[attr.name] = attr.value;
          }

          scriptsData.unshift(scriptData);

          const regex = new RegExp(
            script.outerHTML.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'),
            'g',
          );
          htmlWithMarkers = htmlWithMarkers.replace(regex, marker);
        }

        container.innerHTML = htmlWithMarkers;

        // Convert relative HTMX URLs to absolute URLs
        const htmxElements = container.querySelectorAll(
          '[hx-get], [hx-post], [hx-put], [hx-patch], [hx-delete]',
        );
        const attrs = ['hx-get', 'hx-post', 'hx-put', 'hx-patch', 'hx-delete'];
        htmxElements.forEach(function (el) {
          attrs.forEach(function (attr) {
            const url = el.getAttribute(attr);
            if (url && url.startsWith('/')) {
              el.setAttribute(attr, baseUrl + url);
            }
          });
        });

        // Step #7: HTML Injection & Script Execution
        let alpineStateScript;
        try {
          for (let scriptIdx = 0; scriptIdx < scriptsData.length; scriptIdx++) {
            if (scriptsData[scriptIdx].id === 'alpine-state') {
              alpineStateScript = scriptsData[scriptIdx];
              const alpineStateScriptElement = document.createElement('script');
              alpineStateScriptElement.id = 'alpine-state-exec';
              Object.keys(alpineStateScript.attributes).forEach(function (
                attrName,
              ) {
                alpineStateScriptElement.setAttribute(
                  attrName,
                  alpineStateScript.attributes[attrName],
                );
              });
              alpineStateScriptElement.textContent = alpineStateScript.content;
              document.head.appendChild(alpineStateScriptElement);
              break;
            }
          }

          const markerComments = findComments(container);

          for (var k = 0; k < markerComments.length; k++) {
            const marker = markerComments[k];
            const markerText = marker.nodeValue.trim();
            const match = markerText.match(/MNM_SCRIPT_MARKER:(\d+):(.+)/);

            console.log(marker);
            if (match && match.length === 3) {
              const scriptIndex = parseInt(match[1], 10);
              const scriptId = match[2].trim();
              // Skip alpine-state if we already processed it
              if (scriptId === 'alpine-state' && alpineStateScript) {
                marker.parentNode.removeChild(marker);
                continue;
              }

              if (scriptIndex >= 0 && scriptIndex < scriptsData.length) {
                var scriptData = scriptsData[scriptIndex];

                try {
                  const newScript = document.createElement('script');

                  Object.keys(scriptData.attributes).forEach(function (
                    attrName,
                  ) {
                    newScript.setAttribute(
                      attrName,
                      scriptData.attributes[attrName],
                    );
                  });

                  if (scriptData.src) {
                    newScript.src = scriptData.src;
                  } else {
                    newScript.textContent = scriptData.content;
                  }

                  marker.parentNode.insertBefore(newScript, marker);
                  marker.parentNode.removeChild(marker);
                } catch (error) {
                  console.log(
                    'MeetNearMe Embed: Error inserting script into DOM',
                    error,
                  );
                }
              }
            }
          }
        } catch (error) {
          console.log(
            'MeetNearMe Embed: Error during HTML parsing and script extraction',
            error,
          );
          insertErrorPartial(
            'MeetNearMe Embed: Error during HTML parsing and script extraction: ' +
              error.message,
          );
        }

        // Step #8: Alpine Store Registration
        try {
          checkStores();
        } catch (error) {
          console.log(
            'MeetNearMe Embed: Error during Alpine store registration',
            error,
          );
          insertErrorPartial(
            'MeetNearMe Embed: Error during Alpine store registration: ' +
              error.message,
          );
        }
      })
      .catch(function (error) {
        console.log('MeetNearMe Embed: Failed to load widget', error);
        insertErrorPartial(
          'MeetNearMe Embed: Failed to load widget. ' +
            error.message +
            ' Please try again later.',
        );
      });
  });
})();
