import * as lib from "./lib.js";

export async function makeUrlItems(bid, items) {
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

export async function makeUrlItem(item) {
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
          await browser.tabs.create({
            url: item.url,
          })
        )
      );
    } else {
      throw `unsupported type: ${item.id.type}`;
    }
  } catch (e) {
    result.error = e.message;
  }

  return result;
}

export function urlItemForTab(bid, t) {
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

export function urlItemForBookmark(bid, o) {
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

export async function allTabItems(bid) {
  return (await lib.tabsFromWindows(await lib.getNonAppWindows())).map(
    o => urlItemForTab(bid, o)
  );
}

export async function allBookmarkItems(bid) {
  return (await chrome.bookmarks.search({}))
    .filter((b) => b.children === undefined)
    .map(o => urlItemForBookmark(bid, o));
}

export async function allHistoryItems(bid) {
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

export async function removeUrlItems(bid, items) {
  if (items === undefined) {
    return [];
  }

  let promises = [];

  let results = items.filter((item) => {
    if (item.browser !== bid) {
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

  return results;
}
