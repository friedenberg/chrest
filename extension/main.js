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

async function removeBookmarks(urls) {
  let bookmarks = await chrome.bookmarks.search({});
  let removeBookmarks = [];

  await Promise.all(
    bookmarks.reduce((acc, bm) => {
      if (urls.includes(bm.url)) {
        acc.push(chrome.bookmarks.remove(bm.id));
      }

      return acc;
    }, [])
  );
}

Routes["/urls"] = {
  async get(req) {
    return {
      status: 200,
      body: [
        (await tabsFromWindows(await chrome.windows.getAll())).map((o) => ({
          title: o.title,
          type: "tab",
          id: parseInt(o.id),
          windowId: parseInt(o.windowId),
          url: o.url,
          date: new Date(o.lastAccessed),
        })),
        (await chrome.bookmarks.search({})).map((o) => ({
          title: o.title,
          type: "bookmark",
          id: parseInt(o.id),
          url: o.url,
          date: new Date(o.dateAdded),
        })),
        (await chrome.history.search({ text: "" })).map((o) => ({
          title: o.title,
          type: "history",
          id: parseInt(o.id),
          url: o.url,
          date: new Date(o.lastVisitTime),
        })),
      ].flat(),
    };
  },
  async put(req) {
    await Promise.all(
      await removeTabs(req.body.deleted),
      await removeBookmarks(req.body.deleted)
    );

    return {
      status: 202,
    };
  },
};

//  ____           _
// |  _ \ ___  ___| |_ ___  _ __ ___
// | |_) / _ \/ __| __/ _ \| '__/ _ \
// |  _ <  __/\__ \ || (_) | | |  __/
// |_| \_\___||___/\__\___/|_|  \___|
//

Routes["/restore"] = {
  description: "Restores chrome windows and tabs to a specific state",
  async delete(req) {
    await Promise.all(
      (
        await chrome.windows.getAll()
      ).map((w) => {
        return chrome.windows.remove(w.id);
      })
    );

    return {
      status: 204,
    };
  },
  async post(req) {
    const makePromise = async function (body) {
      let tabs = body["tabs"];
      delete body["tabs"];

      let w = await chrome.windows.create(body);

      tabs = tabs.map((t) => {
        t = cleanTabForSave(t);
        t["windowId"] = w.id;
        return t;
      });

      w.tabs = await Promise.all(tabs.map(makeTab));

      return w;
    };

    return {
      status: 201,
      body: await Promise.all(req.body.map((b) => makePromise(b))),
    };
  },
};

//  ____
// / ___|  __ ___   _____
// \___ \ / _` \ \ / / _ \
//  ___) | (_| |\ V /  __/
// |____/ \__,_| \_/ \___|
//

Routes["/save"] = {
  description: "Outputs windows and tabs in a saveable state",
  async get(req) {
    const windows = await chrome.windows.getAll();
    return {
      status: 200,
      body: (await windowsWithTabs(windows)).map(cleanWindowForSave),
    };
  },
};

// __        ___           _
// \ \      / (_)_ __   __| | _____      _____
//  \ \ /\ / /| | '_ \ / _` |/ _ \ \ /\ / / __|
//   \ V  V / | | | | | (_| | (_) \ V  V /\__ \
//    \_/\_/  |_|_| |_|\__,_|\___/ \_/\_/ |___/
//

const cleanWindowForSave = function (w) {
  delete w["alwaysOnTop"];
  delete w["id"];
  delete w["left"];
  delete w["top"];
  delete w["width"];
  delete w["height"];

  w.tabs = w.tabs.map(cleanTabForSave);

  return w;
}

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

Routes["/windows/last-focused/tabs"] = {
  description: `A symbolic link to /windows/[id for the last focused window].`,
  async get(req) {
    return {
      status: 200,
      body: await chrome.tabs.query({ windowId }),
    };
  },
  async post(req) {
    let w = await windowsWithTabs(await chrome.windows.getLastFocused());

    let added_tabs = await Promise.all(req.body.map((url) => makeTab({url: url, windowId: w.id})));

    w.tabs.push(added_tabs);

    return {
      status: 201,
      body: w,
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

const getWindowTabs = async function (windowId) {
}

//  _____     _
// |_   _|_ _| |__  ___
//   | |/ _` | '_ \/ __|
//   | | (_| | |_) \__ \
//   |_|\__,_|_.__/|___/
//

const makeTab = async function (body) {
  return await chrome.tabs.create(body);
};

const cleanTabForSave = function (t) {
  delete t["audible"];
  delete t["autoDiscardable"];
  delete t["discarded"];
  delete t["favIconUrl"];
  delete t["groupId"];
  delete t["height"];
  delete t["highlighted"];
  delete t["id"];
  delete t["incognito"];
  delete t["lastAccessed"];
  delete t["mutedInfo"];
  delete t["openerTabId"];
  delete t["status"];
  delete t["title"];
  delete t["width"];
  delete t["windowId"];
  return t;
};

async function updateTab(body, groupCache, windowCache) {
  const id = body.id;
  delete body.id;

  if (body.windowIdOrName) {
    let windowId = parseInt(body.windowIdOrName);

    if (isNaN(windowId)) {
      windowId = windowCache[body.windowIdOrName];
    }

    if (windowId == -1) {
      await chrome.tabs.remove(id);
    } else {
      if (!windowId) {
        const w = await chrome.windows.create({ tabId: id });
        windowId = w.id;
      } else {
        await chrome.tabs.move(id, {
          index: -1,
          windowId: windowId,
        });
      }

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

  if (body.open) {
    delete body.open;
    body.active = true;
    await openTab(id);
  }

  delete body.open;

  return await chrome.tabs.update(id, body);
}

async function openTab(id) {
  let tab = await chrome.tabs.get(id);
  let windowId = tab.windowId;
  await Promise.all([
    chrome.windows.update(windowId, { focused: true }),
    chrome.tabs.update(id, { active: true }),
  ]);
}

async function removeTabs(urls) {
  let tabs = await tabsFromWindows(await chrome.windows.getAll());

  await chrome.tabs.remove(
    tabs.reduce((acc, o) => {
      if (urls.includes(o.url)) {
        acc.push(o.id);
      }

      return acc;
    }, [])
  );
}

Routes["/tabs"] = {
  description: "Create a new window.",
  usage: 'echo "https://www.google.com" > $0',
  async post(req) {
    if (Array.isArray(req.body)) {
      return {
        status: 201,
        body: await Promise.all(req.body.map((b) => makeTab(b))),
      };
    } else {
      return {
        status: 201,
        body: await makeTab(req.body),
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

Routes["/tabs/#TAB_ID/open"] = {
  async post({ tabId }) {
    await openTab(tabId);

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

//  ____              _                         _
// | __ )  ___   ___ | | ___ __ ___   __ _ _ __| | _____
// |  _ \ / _ \ / _ \| |/ / '_ ` _ \ / _` | '__| |/ / __|
// | |_) | (_) | (_) |   <| | | | | | (_| | |  |   <\__ \
// |____/ \___/ \___/|_|\_\_| |_| |_|\__,_|_|  |_|\_\___/
//

Routes["/bookmarks"] = {
  description: ``,
  async get() {
    return {
      status: 200,
      body: await chrome.bookmarks.getTree(),
    };
  },
};

//  _   _ _     _
// | | | (_)___| |_ ___  _ __ _   _
// | |_| | / __| __/ _ \| '__| | | |
// |  _  | \__ \ || (_) | |  | |_| |
// |_| |_|_|___/\__\___/|_|   \__, |
//                            |___/

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

    if (response == null) {
      response = { status: 204 };
    }
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
