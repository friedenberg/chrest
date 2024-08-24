
import * as lib from "./lib.js";

export async function makeUrlItems(items) {
  if (items === undefined || items === null) {
    return [];
  }

  return await Promise.all(items.map(makeUrlItem));
};

export async function makeUrlItem(item) {
  let result = item;

  if (item.id.type == "bookmark") {
    Object.assign(result, urlItemForBookmark(await browser.bookmarks.create({
      title: item.title,
      url: item.url,
    })));
  } else if (item.id.type == "tab") {
    Object.assign(result, urlItemForTab(await browser.tabs.create({
      url: item.url,
    })));
  } else {
    throw `unsupported type: ${item.id.type}`;
  }

  return result;
};

export function urlItemForTab(t) {
  return {
    title: t.title,
    id: {
      id: t.id.toString(),
      type: "tab",
    },
    windowId: t.windowId.toString(),
    url: t.url,
    date: new Date(t.lastAccessed),
  };
}

export function urlItemForBookmark(o) {
  return {
    title: o.title,
    id: {
      id: o.id.toString(),
      type: "bookmark",
    },
    url: o.url,
    date: new Date(o.dateAdded),
  };
}

export async function allTabItems() {
  return (await lib.tabsFromWindows(await lib.getNonAppWindows())).map(urlItemForTab);
}

export async function allBookmarkItems() {
  return (await chrome.bookmarks.search({})).map(urlItemForBookmark);
}

export async function allHistoryItems() {
  let history = await chrome.history.search({ text: "" });

  return history.map((o) => ({
    title: o.title,
    id: {
      id: o.id.toString(),
      type: "history",
    },
    url: o.url,
    date: new Date(o.lastVisitTime),
  }));
}

export async function removeUrlItems(items) {
  if (items === undefined) {
    return [];
  }

  let promises = [];

  let results = items.filter(
    (item) => {
      if (item.id.type == "bookmark") {
        promises.push(browser.bookmarks.remove(item.id.id));
        return true
      } else if (item.id.type == "tab") {
        promises.push(browser.tabs.remove(parseInt(item.id.id)));
        return true
      } else {
        // TODO find in all urls
        console.log(item)
        return false;
      }
    },
  );

  await Promise.all(promises);

  return results
}

