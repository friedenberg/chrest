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

        $ http --no-stdin --offline localhost/windows | chrest-whitening client | jq
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

# Installation

1. Go to [Chrome extensions](chrome://extensions/)
1. Enable Developer mode (top-right corner)
1. Load-unpacked the [`extension/`](extension) folder from this repo
1. TBA


