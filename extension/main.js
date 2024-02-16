let Routes = {};

async function windowsWithTabs(windowOrWindowList) {
  if (Array.isArray(windowOrWindowList)) {
    const windows = windowOrWindowList;

    return await Promise.all(
      windows.map(async function (w) {
        w["tabs"] = await chrome.tabs.query({ windowId: w["id"] });
        return w;
      })
    );
  } else {
    const w = windowOrWindowList;
    w["tabs"] = await chrome.tabs.query({ windowId: w["id"] });
    return w;
  }
}

Routes["/windows"] = {
  description: "Create a new window.",
  usage: 'echo "https://www.google.com" > $0',
  async post(req) {
    return {
      status: 201,
      body: await chrome.windows.create({ url: req.body.urls }),
    };
  },
  async get(req) {
    const windows = await chrome.windows.getAll();
    return {
      status: 200,
      body: await windowsWithTabs(windows),
    };
  },
};

Routes["/windows/current"] = {
  description: `A symbolic link to /windows/[id for the last focused window].`,
  async get() {
    return {
      status: 200,
      body: await windowsWithTabs(await chrome.windows.getCurrent()),
    };
  },
};

Routes["/windows/last-focused"] = {
  description: `A symbolic link to /windows/[id for the last focused window].`,
  async get() {
    return {
      status: 200,
      body: await windowsWithTabs(await chrome.windows.getLastFocused()),
    };
  },
};

Routes["/windows/#WINDOW_ID"] = {
  async get({ windowId }) {
    return {
      status: 200,
      body: await windowsWithTabs(await chrome.windows.get(windowId)),
    };
  },
  async put(req) {
    return {
      status: 200,
      body: await windowsWithTabs(await chrome.windows.update(req.vars)),
    };
  },
  async delete({ windowId }) {
    await chrome.windows.remove(windowId);

    return {
      status: 204,
    };
  },
};

Routes["/windows/#WINDOW_ID/tabs"] = {
  async get({ windowId }) {
    return {
      status: 200,
      body: await chrome.tabs.query({ windowId }),
    };
  },
};

Routes["/windows/#WINDOW_ID/tabs/#TAB_ID"] = {
  async get({ tabId }) {
    return {
      status: 200,
      body: await chrome.tabs.get(tabId),
    };
  },
  async delete({ tabId }) {
    await chrome.tabs.remove(tabId);

    return {
      status: 204,
    };
  },
};

Routes["/extensions"] = {
  async get() {
    return {
      status: 200,
      body: await chrome.management.getAll(),
    };
  },
};

Routes["/runtime/reload"] = {
  async post() {
    await chrome.runtime.reload();
    return { status: 204 };
  },
};

const stringToUtf8Array = (function () {
  const encoder = new TextEncoder("utf-8");
  return (str) => encoder.encode(str);
})();

Routes["/tabs/create"] = {
  description: "Create a new tab.",
  usage: 'echo "https://www.google.com" > $0',
  async write({ buf }) {
    const url = buf.trim();
    await chrome.tabs.create({ url });
    return { size: stringToUtf8Array(buf).length };
  },
  async truncate() {
    return {};
  },
};

Routes["/tabs/last-focused"] = {
  description: `Represents the most recently focused tab.
It's a symbolic link to the folder /tabs/by-id/[ID of most recently focused tab].`,
  async readlink() {
    const id = (
      await chrome.tabs.query({ active: true, lastFocusedWindow: true })
    )[0].id;
    return { buf: "by-id/" + id };
  },
};

Routes["/bookmarks"] = {
  description: ``,
  async get() {
    return {
      status: 200,
      body: await chrome.bookmarks.getTree(),
    };
  },
};

for (let key in Routes) {
  // /tabs/by-id/#TAB_ID/url.txt -> RegExp \/tabs\/by-id\/(?<int$TAB_ID>[0-9]+)\/url.txt
  Routes[key].__matchVarCount = 0;
  Routes[key].__regex = new RegExp(
    "^" +
      key
        .split("/")
        .map((keySegment) =>
          keySegment
            .replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
            .replace(/([#:])([A-Z_]+)/g, (_, sigil, varName) => {
              Routes[key].__matchVarCount++;
              return (
                `(?<${sigil === "#" ? "int$" : "string$"}${varName}>` +
                (sigil === "#" ? "[0-9]+" : "[^/]+") +
                `)`
              );
            })
        )
        .join("/") +
      "$"
  );

  Routes[key].__match = function (path) {
    const result = Routes[key].__regex.exec(path);
    if (!result) {
      return;
    }

    const vars = {};
    for (let [typeAndVarName, value] of Object.entries(result.groups || {})) {
      let [type_, varName] = typeAndVarName.split("$");
      // TAB_ID -> tabId
      varName = varName.toLowerCase();
      varName = varName.replace(/_([a-z])/g, (c) => c[1].toUpperCase());
      vars[varName] = type_ === "int" ? parseInt(value) : value;
    }
    return vars;
  };
}

// most specific (lowest matchVarCount) routes should match first
const sortedRoutes = Object.values(Routes).sort(
  (a, b) => a.__matchVarCount - b.__matchVarCount
);

async function tryMatchRoute(req) {
  for (let route of sortedRoutes) {
    const vars = route.__match(req.path);

    if (!vars) {
      continue;
    }

    const routeFunc = route[req.method.toLowerCase()];

    return await routeFunc({ ...req, ...vars });
  }

  return { status: 404 };
}

let port;

async function onMessage(req) {
  let response = {};
  let didTimeout = false,
    timeout = setTimeout(() => {
      // timeout is very useful because some operations just hang
      // (like trying to take a screenshot, until the tab is focused)
      didTimeout = true;
      console.error("timeout");
      port.postMessage({ status: 500, body: { error: unix.ETIMEDOUT } });
    }, 1000);

  try {
    response = await tryMatchRoute(req);
  } catch (e) {
    console.error(e);
    response.body = {
      error: e,
    };
  }

  if (!didTimeout) {
    clearTimeout(timeout);
    port.postMessage(response);
  }
}

function tryConnect() {
  port = chrome.runtime.connectNative("com.linenisgreat.code.chrest");
  port.onMessage.addListener(onMessage);
  port.onDisconnect.addListener((p) => {
    console.log("disconnect", p);
  });
}

if (typeof process === "object") {
  // we're running in node (as part of a test)
  // return everything they might want to test
  module.exports = { Routes, tryMatchRoute };
} else {
  tryConnect();
}
