
export async function windowsWithTabs(windowOrWindowList) {
  if (Array.isArray(windowOrWindowList)) {
    const windows = windowOrWindowList;

    return await Promise.all(
      windows.map(async function(w) {
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

export async function tabsFromWindows(windowOrWindowList) {
  if (Array.isArray(windowOrWindowList)) {
    const windows = windowOrWindowList;

    return (
      await Promise.all(
        windows.map(async function(w) {
          return await chrome.tabs.query({ windowId: w["id"] });
        })
      )
    ).flat();
  } else {
    const w = windowOrWindowList;
    return await chrome.tabs.query({ windowId: w["id"] });
  }
}

export async function removeBookmarks(urls) {
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

export const cleanWindowForSave = function(w) {
  delete w["alwaysOnTop"];
  delete w["id"];
  delete w["left"];
  delete w["top"];
  delete w["width"];
  delete w["height"];

  w.tabs = w.tabs.map(cleanTabForSave);

  return w;
};

export async function normalizeWindowID(windowID) {
  if (windowID === "last-focused") {
    let w = await chrome.windows.getLastFocused();
    windowID = w.id;
  }

  return windowID;
}

export async function getWindowWithID(windowID) {
  var w;

  if (windowID === "last-focused") {
    w = chrome.windows.getLastFocused();
  } else {
    w = chrome.windows.get(windowID);
  }

  return await windowsWithTabs(await w);
}

export const makeTab = async function(body) {
  return await chrome.tabs.create(body);
};

export const makeTabWithWindowId = async function(body, wid) {
  body.windowId = wid;
  return makeTab(body);
};

export const cleanTabForSave = function(t) {
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

export async function updateTab(body, groupCache, windowCache) {
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

export async function openTab(id) {
  let tab = await chrome.tabs.get(id);
  let windowId = tab.windowId;
  await Promise.all([
    chrome.windows.update(windowId, { focused: true }),
    chrome.tabs.update(id, { active: true }),
  ]);
}

export async function removeTabs(urls) {
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

export const stringToUtf8Array = (function() {
  const encoder = new TextEncoder("utf-8");
  return (str) => encoder.encode(str);
})();

