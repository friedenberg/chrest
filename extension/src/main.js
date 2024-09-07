import * as routes from "./routes.js";
import { parse } from 'error-stack-parser-es';

async function tryMatchRoute(req) {
  for (let route of routes.sortedRoutes) {
    const vars = route.__match(req.path);

    if (!vars) {
      continue;
    }

    const routeFunc = route[req.method.toLowerCase()];

    if (!routeFunc) {
      return { status: 404 };
    }

    return await routeFunc({ ...req, ...vars });
  }

  return { status: 404 };
}

let port;
const now = new Date();

async function onMessage(req, messageSender) {
  req.message_sender = messageSender;
  await onMessageHTTP(req);
}

async function onMessageHTTP(req) {
  let response = await Promise.race([
    timeout(1000),
    runRoute(req),
  ]);

  response.headers = {
    "X-Chrest-Startup-Time": now.toISOString(),
    "X-Chrest-UserAgent": Navigator.userAgent,
  };

  response.type = "http";

  console.log(response);
  port.postMessage(response);
}

async function runRoute(req) {
  return tryMatchRoute(req).catch((e) => {
    console.error(e);
    return {
      status: 500,
      body: {
        error: {
          message: e.message,
          stack: parse(e),
        },
      },
    };
  });
}

async function timeout(delay) {
  let timeoutResponse = {
    status: 500,
    body: {
      error: {
        message: "timeout",
      },
    },
  };

  return new Promise(resolve => {
    setTimeout(resolve, delay, timeoutResponse);
  });
}

if (typeof browser == "undefined") {
  // Chrome does not support the browser namespace yet.
  globalThis.browser = chrome;
}

async function initialize(e) {
  browser.storage.sync.onChanged.addListener(changes => {
    console.log(changes);
    let browser_id = changes["browser_id"];

    if (browser_id == undefined) {
      return
    }

    port.postMessage({ type: "who-am-i", browser_id: browser_id.newValue });
  });

  console.log(`try connect: ${e}`);
  port = browser.runtime.connectNative("com.linenisgreat.code.chrest");
  port.onMessage.addListener(onMessage);
  port.onDisconnect.addListener((p) => {
    initialize({ reason: "disconnected" });
  });

  let results = await browser.storage.sync.get("browser_id");

  if (results === undefined || results["browser_id"] === undefined) {
    browser.runtime.openOptionsPage();
  } else {
    let browser_id = results["browser_id"];
    port.postMessage({ type: "who-am-i", browser_id: browser_id, });
  }
}


browser.runtime.onStartup.addListener(() => {
  initialize({ reason: "startup" });
});

browser.runtime.onInstalled.addListener(() => {
  initialize({ reason: "install" });
});
