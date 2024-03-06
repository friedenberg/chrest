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

async function tabsFromWindows(windowOrWindowList) {
  if (Array.isArray(windowOrWindowList)) {
    const windows = windowOrWindowList;

    return (
      await Promise.all(
        windows.map(async function (w) {
          return await chrome.tabs.query({ windowId: w["id"] });
        })
      )
    ).flat();
  } else {
    const w = windowOrWindowList;
    return await chrome.tabs.query({ windowId: w["id"] });
  }
}

async function updateTab(body, groupCache, windowCache) {
  const id = body.id;
  delete body.id;

  // if (body.windowId) {
  //   const windowId = parseInt(body.windowId);
  //   delete body.windowId;

  //   if (windowId == -1) {
  //     return await chrome.tabs.remove(id);
  //   } else {
  //     await chrome.tabs.move(id, { index: -1, windowId: windowId });
  //   }
  // }

  if (body.windowIdOrName) {
    let windowId = parseInt(body.windowIdOrName);

    if (isNaN(windowId)) {
      windowId = windowCache[body.windowIdOrName];
    }

    if (!windowId) {
      const w = await chrome.windows.create();
      windowId = w.id;
    }

    if (windowId == -1) {
      await chrome.tabs.remove(id);
    } else {
      await chrome.tabs.move(id, {
        index: -1,
        windowId: windowId,
      });

      await removeEmptyTabs(windowId);

      windowCache[body.windowIdOrName] = windowId;
    }

    delete body.windowIdOrName;
  }

  if (body.groupIdOrName) {
    let groupId = parseInt(body.groupIdOrName);
    let groupObj = {
      tabIds: id,
    };

    if (isNaN(groupId)) {
      groupId = groupCache[groupIdOrName];
    }

    if (!groupId) {
      groupObj["groupId"] = groupId;
    }

    if (groupId == -1) {
      groupId = await chrome.tabs.ungroup(id);

      groupCache[body.groupIdOrName] = groupId;
    } else {
      groupId = await chrome.tabs.group({
        tabIds: id,
        groupId: groupId,
      });

      groupCache[body.groupIdOrName] = groupId;
    }

    delete body.groupIdOrName;
  }

  return await chrome.tabs.update(id, body);
}

async function removeEmptyTabs(windowId) {
  const tabs = await tabsFromWindows(await chrome.windows.get(windowId));
  console.log(tabs);

  tabs.forEach(
    async function (tab) {
      console.log(tab);
      if (tab.url != "chrome://newtab/") {
        return;
      }

      await chrome.tabs.remove(tab.id);
    }
  );
}

Routes["/bookmarks_and_tabs"] = {
  async get(req) {
    return {
      status: 200,
      body: [
        await tabsFromWindows(await chrome.windows.getAll()),
        await chrome.bookmarks.search({}),
      ].flat(),
    };
  },
};

Routes["/windows"] = {
  description: "Create a new window.",
  usage: 'echo "https://www.google.com" > $0',
  async post(req) {
    const makePromise = async function (body) {
      return await chrome.windows.create(body);
    };

    if (Array.isArray(req.body)) {
      return {
        status: 201,
        body: await Promise.all(req.body.map((b) => makePromise(b))),
      };
    } else {
      return {
        status: 201,
        body: await makePromise(req.body),
      };
    }
  },
  async put(req) {
    const makePromise = async function (body) {
      const id = body.id;
      delete body.id;
      return await chrome.windows.update(id, body);
    };

    if (Array.isArray(req.body)) {
      return {
        status: 200,
        body: await Promise.all(req.body.map((b) => makePromise(b))),
      };
    } else {
      return {
        status: 200,
        body: await makePromise(req.body),
      };
    }
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
      body: await windowsWithTabs(
        await chrome.windows.update(req.windowId, req.body)
      ),
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

Routes["/tabs"] = {
  description: "Create a new window.",
  usage: 'echo "https://www.google.com" > $0',
  async post(req) {
    //TODO
    const makePromise = async function (body) {
      return await chrome.tabs.create(body);
    };

    if (Array.isArray(req.body)) {
      return {
        status: 201,
        body: await Promise.all(req.body.map((b) => makePromise(b))),
      };
    } else {
      return {
        status: 201,
        body: await makePromise(req.body),
      };
    }
  },
  async patch(req) {
    const groupCache = {};
    const windowCache = {};

    if (Array.isArray(req.body)) {
      return {
        status: 200,
        body: await Promise.all(
          req.body.map((b) => updateTab(b, groupCache, windowCache))
        ),
      };
    } else {
      return {
        status: 200,
        body: await updateTab(req.body, groupCache, windowCache),
      };
    }
  },
  async get(req) {
    return {
      status: 200,
      body: await tabsFromWindows(await chrome.windows.getAll()),
    };
  },
};

Routes["/tabs/#TAB_ID"] = {
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
  async patch(req) {
    const groupCache = {};
    const windowCache = {};

    if (Array.isArray(req.body)) {
      return {
        status: 200,
        body: await Promise.all(
          req.body.map((b) => updateTab(b, groupCache, windowCache))
        ),
      };
    } else {
      req.body.id = req.tabId;

      return {
        status: 200,
        body: await updateTab(req.body, groupCache, windowCache),
      };
    }
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

Routes["/history"] = {
  description: ``,
  async get() {
    return {
      status: 200,
      body: await chrome.history.search({ text: "" }),
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

    if (!routeFunc) {
      return { status: 404 };
    }

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
      port.postMessage({ status: 500, body: { error: "timeout" } });
    }, 10000);

  try {
    response = await tryMatchRoute(req);
  } catch (e) {
    console.error(e);
    response.status = 500;
    response.body = {
      error: e.toString(),
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
    tryConnect();
  });
}

if (typeof process === "object") {
  // we're running in node (as part of a test)
  // return everything they might want to test
  module.exports = { Routes, tryMatchRoute };
} else {
  tryConnect();
}
