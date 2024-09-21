'use strict';

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

const cleanWindowForSave = function (w) {
  delete w["alwaysOnTop"];
  delete w["id"];
  delete w["left"];
  delete w["top"];
  delete w["width"];
  delete w["height"];

  w.tabs = w.tabs.map(cleanTabForSave);

  return w;
};

async function normalizeWindowID(windowID) {
  if (windowID === "last-focused") {
    let w = await chrome.windows.getLastFocused();
    windowID = w.id;
  }

  return windowID;
}

async function getNonAppWindows() {
  return (await chrome.windows.getAll()).filter((w) => w["type"] !== "app");
}

async function getWindowWithID(windowID) {
  var w;

  if (windowID === "last-focused") {
    w = chrome.windows.getLastFocused();
  } else {
    w = chrome.windows.get(parseInt(windowID));
  }

  return await windowsWithTabs(await w);
}

const makeTab = async function (body) {
  try {
    const ws = await chrome.windows.getAll();

    if (ws.length == 0) {
      return await chrome.windows.create(body);
    } else {
      return await chrome.tabs.create(body);
    }
  } catch {
    return await chrome.windows.create(body);
  }
};

const makeTabWithWindowId = async function (body, wid) {
  body.windowId = wid;
  return makeTab(body);
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

    if (isNaN(groupId)) {
      groupId = groupCache[groupIdOrName];
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

((function () {
  const encoder = new TextEncoder("utf-8");
  return (str) => encoder.encode(str);
}))();

const FIREFOX_SAFARI_STACK_REGEXP = /(^|@)\S+:\d+/;
const CHROME_IE_STACK_REGEXP = /^\s*at .*(\S+:\d+|\(native\))/m;
const SAFARI_NATIVE_CODE_REGEXP = /^(eval@)?(\[native code\])?$/;
function parse$1(error, options) {
  if (typeof error.stacktrace !== "undefined" || typeof error["opera#sourceloc"] !== "undefined")
    return parseOpera(error, options);
  else if (error.stack && error.stack.match(CHROME_IE_STACK_REGEXP))
    return parseV8OrIE(error, options);
  else if (error.stack)
    return parseFFOrSafari(error, options);
  else if (options?.allowEmpty)
    return [];
  else
    throw new Error("Cannot parse given Error object");
}
function extractLocation(urlLike) {
  if (!urlLike.includes(":"))
    return [urlLike, void 0, void 0];
  const regExp = /(.+?)(?::(\d+))?(?::(\d+))?$/;
  const parts = regExp.exec(urlLike.replace(/[()]/g, ""));
  return [parts[1], parts[2] || void 0, parts[3] || void 0];
}
function applySlice(lines, options) {
  if (options && options.slice != null) {
    if (Array.isArray(options.slice))
      return lines.slice(options.slice[0], options.slice[1]);
    return lines.slice(0, options.slice);
  }
  return lines;
}
function parseV8OrIE(error, options) {
  return parseV8OrIeString(error.stack, options);
}
function parseV8OrIeString(stack, options) {
  const filtered = applySlice(
    stack.split("\n").filter((line) => {
      return !!line.match(CHROME_IE_STACK_REGEXP);
    }),
    options
  );
  return filtered.map((line) => {
    if (line.includes("(eval ")) {
      line = line.replace(/eval code/g, "eval").replace(/(\(eval at [^()]*)|(,.*$)/g, "");
    }
    let sanitizedLine = line.replace(/^\s+/, "").replace(/\(eval code/g, "(").replace(/^.*?\s+/, "");
    const location = sanitizedLine.match(/ (\(.+\)$)/);
    sanitizedLine = location ? sanitizedLine.replace(location[0], "") : sanitizedLine;
    const locationParts = extractLocation(location ? location[1] : sanitizedLine);
    const functionName = location && sanitizedLine || void 0;
    const fileName = ["eval", "<anonymous>"].includes(locationParts[0]) ? void 0 : locationParts[0];
    return {
      function: functionName,
      file: fileName,
      line: locationParts[1] ? +locationParts[1] : void 0,
      col: locationParts[2] ? +locationParts[2] : void 0,
      raw: line
    };
  });
}
function parseFFOrSafari(error, options) {
  return parseFFOrSafariString(error.stack, options);
}
function parseFFOrSafariString(stack, options) {
  const filtered = applySlice(
    stack.split("\n").filter((line) => {
      return !line.match(SAFARI_NATIVE_CODE_REGEXP);
    }),
    options
  );
  return filtered.map((line) => {
    if (line.includes(" > eval"))
      line = line.replace(/ line (\d+)(?: > eval line \d+)* > eval:\d+:\d+/g, ":$1");
    if (!line.includes("@") && !line.includes(":")) {
      return {
        function: line
      };
    } else {
      const functionNameRegex = /(([^\n\r"\u2028\u2029]*".[^\n\r"\u2028\u2029]*"[^\n\r@\u2028\u2029]*(?:@[^\n\r"\u2028\u2029]*"[^\n\r@\u2028\u2029]*)*(?:[\n\r\u2028\u2029][^@]*)?)?[^@]*)@/;
      const matches = line.match(functionNameRegex);
      const functionName = matches && matches[1] ? matches[1] : void 0;
      const locationParts = extractLocation(line.replace(functionNameRegex, ""));
      return {
        function: functionName,
        file: locationParts[0],
        line: locationParts[1] ? +locationParts[1] : void 0,
        col: locationParts[2] ? +locationParts[2] : void 0,
        raw: line
      };
    }
  });
}
function parseOpera(e, options) {
  if (!e.stacktrace || e.message.includes("\n") && e.message.split("\n").length > e.stacktrace.split("\n").length)
    return parseOpera9(e);
  else if (!e.stack)
    return parseOpera10(e);
  else
    return parseOpera11(e, options);
}
function parseOpera9(e, options) {
  const lineRE = /Line (\d+).*script (?:in )?(\S+)/i;
  const lines = e.message.split("\n");
  const result = [];
  for (let i = 2, len = lines.length; i < len; i += 2) {
    const match = lineRE.exec(lines[i]);
    if (match) {
      result.push({
        file: match[2],
        line: +match[1],
        raw: lines[i]
      });
    }
  }
  return applySlice(result, options);
}
function parseOpera10(e, options) {
  const lineRE = /Line (\d+).*script (?:in )?(\S+)(?:: In function (\S+))?$/i;
  const lines = e.stacktrace.split("\n");
  const result = [];
  for (let i = 0, len = lines.length; i < len; i += 2) {
    const match = lineRE.exec(lines[i]);
    if (match) {
      result.push({
        function: match[3] || void 0,
        file: match[2],
        line: match[1] ? +match[1] : void 0,
        raw: lines[i]
      });
    }
  }
  return applySlice(result, options);
}
function parseOpera11(error, options) {
  const filtered = applySlice(
    // @ts-expect-error missing stack property
    error.stack.split("\n").filter((line) => {
      return !!line.match(FIREFOX_SAFARI_STACK_REGEXP) && !line.match(/^Error created at/);
    }),
    options
  );
  return filtered.map((line) => {
    const tokens = line.split("@");
    const locationParts = extractLocation(tokens.pop());
    const functionCall = tokens.shift() || "";
    const functionName = functionCall.replace(/<anonymous function(: (\w+))?>/, "$2").replace(/\([^)]*\)/g, "") || void 0;
    let argsRaw;
    if (functionCall.match(/\(([^)]*)\)/))
      argsRaw = functionCall.replace(/^[^(]+\(([^)]*)\)$/, "$1");
    const args = argsRaw === void 0 || argsRaw === "[arguments not available]" ? void 0 : argsRaw.split(",");
    return {
      function: functionName,
      args,
      file: locationParts[0],
      line: locationParts[1] ? +locationParts[1] : void 0,
      col: locationParts[2] ? +locationParts[2] : void 0,
      raw: line
    };
  });
}

function stackframesLiteToStackframes(liteStackframes) {
  return liteStackframes.map((liteStackframe) => {
    return {
      functionName: liteStackframe.function,
      args: liteStackframe.args,
      fileName: liteStackframe.file,
      lineNumber: liteStackframe.line,
      columnNumber: liteStackframe.col,
      source: liteStackframe.raw
    };
  });
}
function parse(error, options) {
  return stackframesLiteToStackframes(parse$1(error, options));
}

async function makeUrlItems(bid, items) {
  if (items === undefined || items === null) {
    return [];
  }

  try {
    await chrome.windows.getLastFocused();
  } catch (e) {
    await chrome.windows.create();
  }

  return await Promise.all(items.map(o => makeUrlItem(bid, o)));
}

async function makeUrlItem(bid, item) {
  let result = item;

  let itemType = "tab";

  if (item.id !== undefined && item.id.type !== undefined) {
    itemType = item.id.type;
  }

  try {
    if (itemType == "bookmark") {
      Object.assign(
        result,
        urlItemForBookmark(
          bid,
          await browser.bookmarks.create({
            title: item.title,
            url: item.url,
          })
        )
      );
    } else if (itemType == "tab") {
      Object.assign(
        result,
        urlItemForTab(
          bid,
          await browser.tabs.create({
            url: item.url,
          })
        )
      );
    } else {
      throw `unsupported type: ${item.id.type}`;
    }
  } catch (e) {
    console.log(e);
    result.error = {
      message: e.message,
      stack: parse(e),
    };
  }

  return result;
}

function urlItemForTab(bid, t) {
  return {
    title: t.title,
    id: {
      browser: bid,
      id: t.id.toString(),
      type: "tab",
    },
    windowId: t.windowId.toString(),
    url: t.url,
    date: new Date(t.lastAccessed),
  };
}

function urlItemForBookmark(bid, o) {
  return {
    title: o.title,
    id: {
      browser: bid,
      id: o.id.toString(),
      type: "bookmark",
    },
    url: o.url,
    date: new Date(o.dateAdded),
  };
}

async function allTabItems(bid) {
  return (await tabsFromWindows(await getNonAppWindows())).map(
    o => urlItemForTab(bid, o)
  );
}

async function allBookmarkItems(bid) {
  return (await chrome.bookmarks.search({}))
    .filter((b) => b.children === undefined)
    .map(o => urlItemForBookmark(bid, o));
}

async function allHistoryItems(bid) {
  let history = await chrome.history.search({ text: "" });

  return history.map((o) => ({
    title: o.title,
    id: {
      browser: bid,
      id: o.id.toString(),
      type: "history",
    },
    url: o.url,
    date: new Date(o.lastVisitTime),
  }));
}

async function removeUrlItems(bid, items) {
  if (items === undefined) {
    return [];
  }

  let promises = [];

  let results = items.filter((item) => {
    let theirBid = item.id.browser;

    if (theirBid.browser != bid.browser || theirBid.id != bid.id) {
      return false;
    }

    if (item.id.type == "bookmark") {
      promises.push(browser.bookmarks.remove(item.id.id));
      return true;
    } else if (item.id.type == "tab") {
      promises.push(browser.tabs.remove(parseInt(item.id.id)));
      return true;
    } else {
      // TODO find in all urls
      console.log(item);
      return false;
    }
  });

  await Promise.all(promises);
  console.log(promises, results);

  return results;
}

let Routes = {};

Routes["/"] = {
  async get(req) {
    return {
      status: 200,
      body: {
        browser: "firefox",
        platform_info: browser.runtime.getPlatformInfo(),
        request: req,
      },
    };
  },
};

Routes["/items"] = {
  async get(req) {
    return {
      status: 200,
      body: [
        await allTabItems(req.browser_id),
        await allBookmarkItems(req.browser_id),
        await allHistoryItems(req.browser_id),
      ].flat(),
    };
  },
  async put(req) {
    return {
      body: {
        added: await makeUrlItems(req.browser_id, req.body.added),
        deleted: await removeUrlItems(req.browser_id, req.body.deleted),
      },
      status: 200,
    };
  },
};

//  ____  _        _
// / ___|| |_ __ _| |_ ___
// \___ \| __/ _` | __/ _ \
//  ___) | || (_| | ||  __/
// |____/ \__\__,_|\__\___|
//

Routes["/state"] = {
  description:
    "Save, restore, or clear chrome windows and tabs with to a state",
  async get(req) {
    const windows = await chrome.windows.getAll();
    return {
      status: 200,
      body: (await windowsWithTabs(windows)).map(cleanWindowForSave),
    };
  },
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

      if (body.type !== "normal") {
        return null;
      }

      delete body.type;

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
      body: await getWindowWithID(windowId),
    };
  },
  async put(req) {
    let wid = await normalizeWindowID(req.windowId);

    return {
      status: 200,
      body: await windowsWithTabs(
        await chrome.windows.update(wid, req.body)
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
  async post(req) {
    let wid = await normalizeWindowID(req.windowId);

    if (Array.isArray(req.body)) {
      return {
        status: 201,
        body: await Promise.all(
          req.body.map((b) => makeTabWithWindowId(b, wid))
        ),
      };
    } else {
      return {
        status: 201,
        body: await makeTabWithWindowId(req.body, wid),
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
    let wid = await normalizeWindowID(req.windowId);

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
    let wid = await normalizeWindowID(req.windowId);

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
      body: await chrome.tabs.get(parseInt(tabId)),
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
  async get() {
    return {
      status: 200,
      body: await chrome.runtime.getManifest(),
    };
  },
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
      vars[varName] = value;
      // vars[varName] = type_ === "int" ? parseInt(value) : value;
    }
    return vars;
  };
}

// most specific (lowest matchVarCount) routes should match first
const sortedRoutes = Object.values(Routes).sort(
  (a, b) => a.__matchVarCount - b.__matchVarCount
);

var browser_type = "firefox";

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
const now = new Date();

async function onMessage(req, messageSender) {
  req.message_sender = messageSender;
  await onMessageHTTP(req);
}

async function onMessageHTTP(req) {
  let results = await browser.storage.sync.get("browser_id");

  if (results === undefined || results["browser_id"] === undefined) ; else {
    req.browser_id = {
      browser: browser_type,
      id: results["browser_id"],
    };
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
    if (e.reason == "install") {
      browser.runtime.openOptionsPage();
    }
  } else {
    await initializePort(results["browser_id"]);
  }
}

async function initializePort(browser_id) {
  if (port != undefined) {
    port.disconnect();
  }

  console.log(`try connect: ${JSON.stringify(browser_id)}`);
  port = browser.runtime.connectNative("com.linenisgreat.code.chrest");
  port.onMessage.addListener(onMessage);
  // port.onDisconnect.addListener((p) => {
  //   initialize({ reason: "disconnected", error: browser.runtime.lastError });
  // });
  port.postMessage({
    type: "who-am-i",
    browser_id: browserIdFromSettingString(browser_id),
  });
}

browser.runtime.onStartup.addListener(() => {
  initialize({ reason: "startup" });
});

browser.runtime.onInstalled.addListener(() => {
  initialize({ reason: "install" });
});
