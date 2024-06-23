import * as lib from "./lib.js";

export let Routes = {};

Routes["/urls"] = {
  async get(req) {
    return {
      status: 200,
      body: [
        (await lib.tabsFromWindows(await chrome.windows.getAll())).map((o) => ({
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
      await lib.removeTabs(req.body.deleted),
      await lib.removeBookmarks(req.body.deleted)
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
    const makePromise = async function(body) {
      let tabs = body["tabs"];
      delete body["tabs"];

      if (body.type !== "normal") {
        return null;
      }

      delete body.type;

      let w = await chrome.windows.create(body);

      tabs = tabs.map((t) => {
        t = lib.cleanTabForSave(t);
        t["windowId"] = w.id;
        return t;
      });

      w.tabs = await Promise.all(tabs.map(lib.makeTab));

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
      body: (await lib.windowsWithTabs(windows)).map(lib.cleanWindowForSave),
    };
  },
};

// __        ___           _
// \ \      / (_)_ __   __| | _____      _____
//  \ \ /\ / /| | '_ \ / _` |/ _ \ \ /\ / / __|
//   \ V  V / | | | | | (_| | (_) \ V  V /\__ \
//    \_/\_/  |_|_| |_|\__,_|\___/ \_/\_/ |___/
//

Routes["/windows"] = {
  description: "Create a new window.",
  usage: 'echo "https://www.google.com" > $0',
  async post(req) {
    const makePromise = async function(body) {
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
    const makePromise = async function(body) {
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
      body: await lib.windowsWithTabs(windows),
    };
  },
};

Routes["/windows/current"] = {
  description: `A symbolic link to /windows/[id for the last focused window].`,
  async get() {
    return {
      status: 200,
      body: await lib.windowsWithTabs(await chrome.windows.getCurrent()),
    };
  },
};

// Routes["/windows/last-focused"] = {
//   description: `A symbolic link to /windows/[id for the last focused window].`,
//   async get() {
//     return {
//       status: 200,
//       body: await windowsWithTabs(await chrome.windows.getLastFocused()),
//     };
//   },
// };

// Routes["/windows/last-focused/tabs"] = {
//   description: `A symbolic link to /windows/[id for the last focused window].`,
//   async get(req) {
//     return {
//       status: 200,
//       body: await chrome.tabs.query({ windowId }),
//     };
//   },
//   async post(req) {
//     let w = await windowsWithTabs(await chrome.windows.getLastFocused());

//     let added_tabs = await Promise.all(
//       req.body.map((url) => makeTab({ url: url, windowId: w.id }))
//     );

//     w.tabs.push(added_tabs);

//     return {
//       status: 201,
//       body: w,
//     };
//   },
// };
Routes["/windows/#WINDOW_ID"] = {
  async get({ windowId }) {
    return {
      status: 200,
      body: await lib.getWindowWithID(windowId),
    };
  },
  async put(req) {
    let wid = await lib.normalizeWindowID(req.windowId);

    return {
      status: 200,
      body: await lib.windowsWithTabs(await chrome.windows.update(wid, req.body)),
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
  async post(req) {
    let wid = await lib.normalizeWindowID(req.windowId);

    if (Array.isArray(req.body)) {
      return {
        status: 201,
        body: await Promise.all(req.body.map(b => (lib.makeTabWithWindowId(b, wid)))),
      };
    } else {
      return {
        status: 201,
        body: await lib.makeTabWithWindowId(req.body, wid),
      };
    }
  },
};

Routes["/windows/#WINDOW_ID/tab-urls"] = {
  async get({ windowId }) {
    return {
      status: 200,
      body: (await chrome.tabs.query({ windowId })).map((t) => t.url),
    };
  },
  async post(req) {
    let wid = await lib.normalizeWindowID(req.windowId);

    await Promise.all(
      req.body.map((url) => {
        return chrome.tabs.create({ windowId: wid, url: url });
      })
    );

    return {
      status: 200,
    };
  },
  async post(req) {
    let wid = await lib.normalizeWindowID(req.windowId);

    await Promise.all(
      req.body.map((url) => {
        return chrome.tabs.create({ windowId: wid, url: url });
      })
    );

    return {
      status: 200,
    };
  },
};

//  _____     _
// |_   _|_ _| |__  ___
//   | |/ _` | '_ \/ __|
//   | | (_| | |_) \__ \
//   |_|\__,_|_.__/|___/
//


Routes["/tabs"] = {
  description: "Create a new window.",
  usage: 'echo "https://www.google.com" > $0',
  async post(req) {
    if (Array.isArray(req.body)) {
      return {
        status: 201,
        body: await Promise.all(req.body.map((b) => lib.makeTab(b))),
      };
    } else {
      return {
        status: 201,
        body: await lib.makeTab(req.body),
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
          req.body.map((b) => lib.updateTab(b, groupCache, windowCache))
        ),
      };
    } else {
      return {
        status: 200,
        body: await lib.updateTab(req.body, groupCache, windowCache),
      };
    }
  },
  async get(req) {
    return {
      status: 200,
      body: await lib.tabsFromWindows(await chrome.windows.getAll()),
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
          req.body.map((b) => lib.updateTab(b, groupCache, windowCache))
        ),
      };
    } else {
      // req.body.id = req.tabId;
      let res = await chrome.tabs.update(parseInt(req.tabId), req.body);

      return {
        status: 200,
        body: res,
      };
    }
  },
};

Routes["/tabs/#TAB_ID/open"] = {
  async post({ tabId }) {
    await lib.openTab(tabId);

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
            console.log(key, sigil, varName);
            Routes[key].__matchVarCount++;
            return (
              `(?<${sigil === "#" ? "int$" : "string$"}${varName}>` +
              (sigil === "#" ? "[^/]+" : "[^/]+") +
              `)`
            );
          })
      )
      .join("/") +
    "$"
  );

  Routes[key].__match = function(path) {
    console.log(path);
    const result = Routes[key].__regex.exec(path);
    if (!result) {
      return;
    }

    console.log(key);

    const vars = {};
    for (let [typeAndVarName, value] of Object.entries(result.groups || {})) {
      let [type_, varName] = typeAndVarName.split("$");
      // TAB_ID -> tabId
      varName = varName.toLowerCase();
      varName = varName.replace(/_([a-z])/g, (c) => c[1].toUpperCase());
      vars[varName] = value;
      // vars[varName] = type_ === "int" ? parseInt(value) : value;
    }
    return vars;
  };
}

// most specific (lowest matchVarCount) routes should match first
export const sortedRoutes = Object.values(Routes).sort(
  (a, b) => a.__matchVarCount - b.__matchVarCount
);

