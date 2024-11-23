import * as routes from "./routes.js";
import { parse } from "error-stack-parser-es";
import { Mutex } from "async-mutex";
import browser_type from 'consts:browser_type';

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
const mutex = new Mutex();

async function onMessage(req, messageSender) {
  req.message_sender = messageSender;
  await mutex.runExclusive(async () => onMessageHTTP(req));
}

async function onMessageHTTP(req) {
  let results = await browser.storage.sync.get("browser_id");

  if (results === undefined || results["browser_id"] === undefined) {
    // TODO ERROR
  } else {
    req.browser_id = {
      browser: browser_type,
      id: results["browser_id"],
    }
  }

  let response = await Promise.race([timeout(1000), runRoute(req)]);

  response.headers = {
    "X-Chrest-Startup-Time": now.toISOString(),
    "X-Chrest-Browser-Type": browser_type,
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

  return new Promise((resolve) => {
    setTimeout(resolve, delay, timeoutResponse);
  });
}

if (typeof browser == "undefined") {
  // Chrome does not support the browser namespace yet.
  globalThis.browser = chrome;
}

function browserIdFromSettingString(v) {
  return `${browser_type}-${v}`;
}

async function initialize(e) {
  browser.storage.sync.onChanged.addListener((changes) => {
    console.log(changes);
    let browser_id = changes["browser_id"];

    if (browser_id == undefined) {
      return;
    }

    initializePort(browser_id.newValue);
  });

  let results = await browser.storage.sync.get("browser_id");

  if (results === undefined || results["browser_id"] === undefined) {
    browser.runtime.openOptionsPage();
  } else {
    await initializePort(results["browser_id"]);
  }
}

async function deinitializePort() {
  if (port != undefined) {
    port.disconnect();
  }
}

async function initializePort(browser_id) {
  await deinitializePort();

  console.log(`try connect: ${JSON.stringify(browser_id)}`);
  port = browser.runtime.connectNative("com.linenisgreat.code.chrest");
  port.onMessage.addListener(onMessage);
  port.postMessage({
    type: "who-am-i",
    browser_id: browserIdFromSettingString(browser_id),
  });
}

// browser.runtime.onStartup.addListener(() => {
//   initialize({ reason: "startup" });
// });

// browser.runtime.onInstalled.addListener(() => {
//   initialize({ reason: "install" });
// });

initialize({ reason: "startup" });
