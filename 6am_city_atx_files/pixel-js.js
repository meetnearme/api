(function () {
  'use strict';

  (function() {
    if (isCrawler()) {
      return;
    }
    let log = () => {
    };
    const APIARY_ENDPOINT = "https://ingestion.staging.apiarydata.net/api/v1/ingestion/pixel";
    const EXCLUDED_DOMAINS = ["beehiiv.com", "staginghiiv.com", "localhiiv.com"];
    let isSecure = true;
    if (APIARY_ENDPOINT.includes("dev")) {
      log = console.log;
      isSecure = false;
    }
    try {
      log("pixel-js");
      let [ad_network_placement_id, subscriber_id, event, bhp] = get_bhcl_id();
      sendInitialEvent(event, ad_network_placement_id, subscriber_id, bhp);
    } catch (error) {
      console.error(error);
    }
    function sendInitialEvent(event, ad_network_placement_id, subscriber_id, bhp) {
      if (!ad_network_placement_id) return;
      sendEvent(event, ad_network_placement_id, subscriber_id, bhp);
      monitorUrlChanges(() => {
        sendEvent("pageview", ad_network_placement_id, subscriber_id, bhp);
      });
    }
    function monitorUrlChanges(onUrlChange) {
      const originalPushState = history.pushState;
      const originalReplaceState = history.replaceState;
      function triggerUrlChangeEvent() {
        const event = new Event("bhpx:urlchange");
        window.dispatchEvent(event);
      }
      history.pushState = function(...args) {
        originalPushState.apply(this, args);
        triggerUrlChangeEvent();
      };
      history.replaceState = function(...args) {
        originalReplaceState.apply(this, args);
        triggerUrlChangeEvent();
      };
      window.addEventListener("popstate", () => {
        triggerUrlChangeEvent();
      });
      window.addEventListener("bhpx:urlchange", onUrlChange);
    }
    function get_bhcl_id(options) {
      options = options || {};
      let { host, domain } = getHostDomain();
      let event = "pageview";
      const urlParams = new URLSearchParams(window.location.search);
      let bhcl_id = urlParams.get("bhcl_id");
      if (options.host) {
        log(`using custom host: ${options.host}`);
        host = options.host;
      }
      const allCookies = document.cookie.split(";");
      const [bhcl, cookieName] = findCookieWithHost(allCookies, "_bhcl", host, domain);
      let bhp = getCookieValue(findCookie(allCookies, "_bhp"));
      if (!bhp) {
        bhp = generateUUID();
        updateCookie("_bhp", bhp, domain);
      }
      let cookie_bhcl_id = bhcl ? getCookieValue(bhcl) : "";
      if (bhcl_id && bhcl_id !== cookie_bhcl_id) {
        event = "first_visited";
      } else if (cookie_bhcl_id) {
        bhcl_id = cookie_bhcl_id;
        log(`bhcl_id found in cookie: ${cookieName}`, bhcl_id);
      }
      if (bhcl_id) {
        updateBhclCookie(cookieName, bhcl_id, domain);
      }
      if (!bhcl_id) {
        log("no bhcl_id found");
        return [];
      }
      let [ad_network_placement_id, subscriber_id] = bhcl_id.split("_");
      if (subscriber_id?.match(/[^0-9a-f-]/)) {
        log(`invalid subscriber_id ${subscriber_id}`);
        subscriber_id = "";
      }
      return [ad_network_placement_id, subscriber_id, event, bhp];
    }
    function getHostDomain() {
      const { hostname } = window.location;
      if (hostname === "localhost" || hostname === "127.0.0.1") return [];
      let host = "www";
      let domain = "";
      const parts = hostname.split(".");
      if (parts.length < 3) {
        domain = `${parts[0]}.${parts[1]}`;
      } else {
        host = parts[0];
        domain = `${parts[1]}.${parts[2]}`;
      }
      return { host, domain };
    }
    function updateBhclCookie(name, value, domain) {
      const isExcludedDomain = EXCLUDED_DOMAINS.includes(domain);
      if (!isExcludedDomain) {
        if (name !== "_bhcl") {
          removeCookie(name, domain);
        }
        name = "_bhcl";
      }
      updateCookie(name, value, domain);
      log(`bhcl_id added to cookie: ${name}`);
    }
    function updateCookie(name, value, domain) {
      const expires = 365 * 24 * 60 * 60;
      const cookieProps = `domain=.${domain}; path=/; samesite=strict; ${isSecure ? "secure;" : ""} max-age=${expires}`;
      document.cookie = `${name}=${value}; ${cookieProps}`;
    }
    function removeCookie(name, domain) {
      const expires = 0;
      const value = "";
      const cookieProps = `domain=.${domain}; path=/; samesite=strict; ${isSecure ? "secure;" : ""} max-age=${expires}`;
      document.cookie = `${name}=${value}; ${cookieProps}`;
    }
    function findCookie(allCookies, name) {
      return allCookies.find((cookie) => cookie.trim().startsWith(`${name}=`));
    }
    function findCookieWithHost(allCookies, name, host, domain) {
      let cookie;
      const isExcludedDomain = EXCLUDED_DOMAINS.includes(domain);
      if (!isExcludedDomain) {
        cookie = findCookie(allCookies, name);
      }
      if (!cookie) {
        name = `${name}_${host}`;
        cookie = findCookie(allCookies, name);
      }
      if (!cookie && host !== "www") {
        name = `${name}_www`;
        cookie = findCookie(allCookies, name);
      }
      return [cookie, name];
    }
    function getCookieValue(cookie) {
      return cookie ? cookie.split("=")[1] : "";
    }
    function sendEvent(event, ad_network_placement_id, subscriber_id, bhp, data) {
      if (!event || !ad_network_placement_id) return;
      const event_id = generateUUID();
      const timestamp = (/* @__PURE__ */ new Date()).getTime();
      const {
        content_category,
        content_ids,
        content_name,
        content_type,
        currency,
        num_items,
        predicted_ltv_cents,
        search_string,
        status,
        value_cents
      } = data || {};
      const payload = {
        ad_network_placement_id,
        subscriber_id: subscriber_id ?? "",
        profile_id: bhp ?? "",
        // anonymous profile id
        event,
        timestamp,
        landed_timestamp: timestamp,
        sent_timestamp: timestamp,
        event_id,
        url: window.location.href,
        user_agent: window.navigator.userAgent,
        // custom data properties are optional
        content_category,
        content_ids,
        content_name,
        content_type,
        currency,
        num_items,
        predicted_ltv_cents: getInt(predicted_ltv_cents),
        search_string,
        status,
        value_cents: getInt(value_cents)
      };
      log(`sending ${event} event to pixel endpoint`, JSON.parse(JSON.stringify(payload)));
      fetch(APIARY_ENDPOINT, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify([payload])
        // payload as array
      }).then((response) => {
        log("response", response.ok, response.status, response.statusText);
      }).catch((error) => {
        console.error(error);
      });
    }
    function isCrawler() {
      const ua = navigator.userAgent.toLowerCase();
      const crawlerRegex = /(bot|crawl|spider|slurp|archiver|indexer|facebookexternalhit|twitterbot|bingpreview|applebot|siteaudit|semrush|ahrefs|mj12bot|seznambot|screaming frog|dotbot)/i;
      return crawlerRegex.test(ua);
    }
    function generateUUID() {
      const arr = new Uint8Array(16);
      window.crypto.getRandomValues(arr);
      arr[6] = arr[6] & 15 | 64;
      arr[8] = arr[8] & 63 | 128;
      return [...arr].map((b, i) => {
        const hex = b.toString(16).padStart(2, "0");
        return i === 4 || i === 6 || i === 8 || i === 10 ? `-${hex}` : hex;
      }).join("");
    }
    function getInt(s) {
      if (typeof s === "number") return s;
      if (typeof s === "string") return parseInt(s, 10);
      return void 0;
    }
    window.bhpx = function(command, event, options) {
      const [ad_network_placement_id, subscriber_id, _event, bhp] = get_bhcl_id(options);
      const { data } = options || {};
      switch (command) {
        case "track":
          const valid_events = [
            "conversion",
            "lead",
            "complete_registration",
            "purchase",
            "initiate_checkout",
            "start_trial",
            "subscribe"
          ];
          if (!valid_events.includes(event)) {
            console.error("bhpx: invalid event", event);
            return;
          }
          sendEvent(event, ad_network_placement_id, subscriber_id, bhp, data);
          break;
        default:
          console.error("bhpx: unknown command", command);
      }
    };
  })();

})();
