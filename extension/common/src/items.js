import * as lib from "./lib.js";
import { parse } from "error-stack-parser-es";

export async function makeUrlItems(bid, items) {
  if (items === undefined || items === null) {
    return [];
  }

  try {
    await browser.windows.getLastFocused();
  } catch (e) {
    await browser.windows.create();
  }

  return await Promise.all(items.map(o => makeUrlItem(bid, o)));
}

export async function makeUrlItem(bid, item) {
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
      let tab = await browser.tabs.create({ url: item.url });

      Object.assign(
        result,
        urlItemForTab(bid, tab,)
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
  return (await browser.bookmarks.search({}))
    .filter((b) => b.type === "bookmark")
    .map(o => urlItemForBookmark(bid, o));
}

export async function allHistoryItems(bid) {
  let history = await browser.history.search({ text: "" });

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
    let theirBid = item.id.browser;

    if (theirBid.browser != bid.browser || theirBid.id != bid.id) {
      return false;
    }

    try {
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
    } catch (e) {
      console.log("failed to remove item", item);
      return false;
    }
  });

  await Promise.all(promises);
  console.log(promises, results);

  return results;
}
