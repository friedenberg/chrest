Chrest is a CLI tool and Chrome extension that allow you to manage Chrome via
REST. Chrest was inspired by (and somewhat forked from)
[`TabFS`](https://omar.website/tabfs/).

Chrest consists of three parts:

1.  Chrome Extension: `Chrest` ([/extension](extension))
2.  Native Messaging Host: `chrest-cavity` ([/go/cavity](go/cavity))
3.  CLI Client: `chrest-whitening` ([/go/whitening](go/whitening))

# Examples

Note: to keep Chrest slim, Chrest does not include pretty-print JSON or HTTP
Request construction, but there are other wonderful tools you can use for that
like [`http`](https://httpie.io/) and [`jq`](https://jqlang.github.io/jq/).

-   Getting all windows and tabs:

        $ http --ignore-stdin --offline localhost/windows |
        $   chrest-whitening client |
        $   jq
        > [
        >   {
        >     "alwaysOnTop": false,
        >     "focused": false,
        >     "height": 1400,
        >     "id": 1531273373,
        >     "incognito": false,
        >     "left": 1411,
        >     "state": "normal",
        >     "tabs": [
        >       {
        >         "active": true,
        >         "audible": false,
        >         "autoDiscardable": true,
        >         "discarded": false,
        >         "favIconUrl": "https://github.githubassets.com/favicons/favicon.svg",
        >         "groupId": -1,
        >         "height": 1279,
        >         "highlighted": true,
        >         "id": 1531273367,
        >         "incognito": false,
        >         "index": 0,
        >         "mutedInfo": {
        >           "muted": false
        >         },
        >         "pinned": false,
        >         "selected": true,
        >         "status": "complete",
        >         "title": "friedenberg/chrest",
        >         "url": "https://github.com/friedenberg/chrest",
        >         "width": 1128,
        >         "windowId": 1531273373
        >       }
        >     ],
        >     "top": 20,
        >     "type": "normal",
        >     "width": 1128
        >   }
        > ]

-   Creating a new window with some URL's:

        $ http --ignore-stdin --offline localhost/windows \
        $   urls[]=https://github.com/friedenberg/chrest |
        $   chrest-whitening client |
        $   jq
        > {
        >   "alwaysOnTop": false,
        >   "focused": false,
        >   "height": 1400,
        >   "id": 1531273390,
        >   "incognito": false,
        >   "left": 0,
        >   "state": "normal",
        >   "tabs": [
        >     {
        >       "active": true,
        >       "audible": false,
        >       "autoDiscardable": true,
        >       "discarded": false,
        >       "groupId": -1,
        >       "height": 1279,
        >       "highlighted": true,
        >       "id": 1531273391,
        >       "incognito": false,
        >       "index": 0,
        >       "lastAccessed": 1708099951989.452,
        >       "mutedInfo": {
        >         "muted": false
        >       },
        >       "pendingUrl": "https://github.com/friedenberg/chrest",
        >       "pinned": false,
        >       "selected": true,
        >       "status": "loading",
        >       "title": "",
        >       "url": "",
        >       "width": 1128,
        >       "windowId": 1531273390
        >     }
        >   ],
        >   "top": 0,
        >   "type": "normal",
        >   "width": 1128
        > }

-   Closing a window:

        $ http --ignore-stdin --offline DELETE localhost/windows/1531273390 |
        $   chrest-whitening client |
        $   jq

# Installation

1.  Go to [Chrome extensions](chrome://extensions/)
2.  Enable Developer mode (top-right corner)
3.  Load-unpacked the [`extension/`](extension) folder from this repo
4.  TBA

# Todo

-   mutate bookmark endpoints
-   proper `PUT` endpoints for windows and tabs
-   history endpoints

# Known Issues

-   Chrome will turn off service workers when there are no windows left, so at
    least 1 window needs to exist for `chrest-cavity` to be running. I may add a
    fallback in `chrest-whitening` to use `AppleScript` to create a new window
    if the server is not running.
