import * as routes from "./routes.js";
import { parse } from 'stack-trace';

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
  let response = {};
  let didTimeout = false,
    timeout = setTimeout(() => {
      // timeout is very useful because some operations just hang
      // (like trying to take a screenshot, until the tab is focused)
      didTimeout = true;
      console.error("timeout");
      port.postMessage({ status: 500, body: { error: "timeout" } });
    }, 10000);

  try {
    response = await tryMatchRoute(req);

    if (response == null) {
      response = { status: 204 };
    }
  } catch (e) {
    console.error(e);
    response.status = 500;
    response.body = {
      error: {
        message: e.message,
        stack: parse(e),
      },
    };
  }

  if (!didTimeout) {
    clearTimeout(timeout);
    response.headers = {
      "X-Chrest-Startup-Time": now.toISOString(), 
    };
    port.postMessage(response);
  }
}

function tryConnect(e) {
  console.log(`try connect: ${e}`);
  port = chrome.runtime.connectNative("com.linenisgreat.code.chrest");
  port.onMessage.addListener(onMessage);
  port.onDisconnect.addListener((p) => {
    console.log("disconnect", p);
    tryConnect();
  });
}

if (typeof process === "object") {
  // we're running in node (as part of a test)
  // return everything they might want to test
  module.exports = { Routes, tryMatchRoute };
} else {
  tryConnect(null);
}

chrome.runtime.onStartup.addListener(() => {
  tryConnect({ reason: "startup" });
});

// chrome.runtime.onInstalled.addListener(tryConnect);
