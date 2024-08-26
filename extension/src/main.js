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

async function onMessage(req) {
  let response = await Promise.race([
    timeout(1000),
    runRoute(req),
  ]);

  response.headers = {
    "X-Chrest-Startup-Time": now.toISOString(),
    "X-Chrest-UserAgent": Navigator.userAgent,
  };

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

function tryConnect(e) {
  console.log(`try connect: ${e}`);
  port = browser.runtime.connectNative("com.linenisgreat.code.chrest");
  port.onMessage.addListener(onMessage);
  port.onDisconnect.addListener((p) => {
    console.log("disconnect", p);
  });
}

if (typeof browser == "undefined") {
  // Chrome does not support the browser namespace yet.
  globalThis.browser = chrome;
}

browser.runtime.onStartup.addListener(() => {
  tryConnect({ reason: "startup" });
});

browser.runtime.onInstalled.addListener(() => {
  tryConnect({ reason: "install" });
});
