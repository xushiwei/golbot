## golbot - A Lua scriptable chat bot

golbot is a Lua scriptable chat bot written in Go. 

## Supported procotols

- IRC
- Slack
- Hipchat
- RocketChat

## Install

```bash
go get github.com/yuin/golbot
```

## Getting started

```bash
golbot init PROTOCOL # golbot.lua will be generated
golbot run
```

## Scripting with Lua

Here is a default config script for IRC :

```lua
local golbot = require("golbot")                           -- 1
local json = require("json")
local charset = require("charset")
-- local re = require("re")
-- local requests = require("requests")
-- local sh = require("sh")
-- local fs = require("fs")

function main()
  mynick = "golbot"
  myname = "golbot"
  msglog = golbot.newlogger({"seelog", type="adaptive", mininterval="200000000", maxinterval="1000000000", critmsgcount="5",
      {"formats",
        {"format", id="main", format="%Date(2006-01-02 15:04:05) %Msg"},
      },
      {"outputs", formatid="main",
        {"filter", levels="trace,debug,info,warn,error,critical",
          {"console"}
        },
      }
  })

  local bot = golbot.newbot("IRC", {                       -- 2
    nickname = mynick,
    username = myname,
    conn = "localhost:6667,#test",
    useTLS = false,
    password = "password",
    http = "0.0.0.0:6669",
    log = {"seelog", type="adaptive", mininterval="200000000", maxinterval="1000000000", critmsgcount="5",
      {"formats",
        {"format", id="main", format="%Date(2006-01-02 15:04:05) [%Level] %Msg"},
      },
      {"outputs", formatid="main",
        {"filter", levels="trace,debug,info,warn,error,critical",
          {"console"}
        },
      }
    }
  })

  bot:respond([[\s*(\d+)\s*\+\s*(\d+)\s*]], function(m, e) -- 3
    bot:say(e.target, tostring(tonumber(m[2]) + tonumber(m[3])))
  end)

  bot:on("PRIVMSG", function(e)                            -- 4
    local ch = e.arguments[1]
    local nick = e.nick
    local user = e.user
    local source = e.source
    local msg = e:message()
    if nick == mynick then
      return
    end

    msglog:printf("%s\t%s\t%s", ch, source, msg)
    bot.raw:privmsg(ch, msg)                               -- 5
    goworker({channel=ch, message=msg, nick=nick})         -- 6
  end)

  bot:serve(function(msg)                                  -- 7
    if msg.type == "say" then
      bot:say(msg.channel, msg.message)
      respond(msg, true)                                   -- 8
    end
  end)
end

function worker(msg)                                       -- 9
  notifymain({type="say", channel=msg.channel, message="accepted"}) -- 10
end

function http(r)                                           -- 11
  if r.method == "POST" and r.URL.path == "/say" then
    local msg = json.decode(r:readbody())
    local ok, success = requestmain({type="say", channel=msg.channel, message=msg.message) -- 12
    if ok and success then
      return 200,
             {
               {"Content-Type", "application/json; charset=utf-8"}
             },
             json.encode({result="ok"})
    else
      return 406,
             {
               {"Content-Type", "application/json; charset=utf-8"}
             },
             json.encode({result="error"})
    end
  end
  return 400,
         {
           {"Content-Type", "text/plain; charset=utf-8"}
         },
         "NOT FOUND"
end
```

- 1. requires golbot library.
- 2. creates new bot.
    - `#1` : chat type. currently supports `"IRC"`, `"Slack"`, `"Hipchat"` and `"Rocket"`
    - `#2` : options(including protocol specific) as a table 
        - Common options are:
            - `log` : 
                - `function` : function to log system messages( `function(msg:string) end` )
                - `table` : [seelog](https://github.com/cihub/seelog) XML configuration as a lua table to log system messages
            - `http` : Address with port for binding HTTP REST API server
        - `nickname`, `username`, `conn`, `userTLS` and `password` are IRC specific options
- 3. adds a callback that will be called when bot receives a message. `respond` will be called only when the message contains a mention to the bot.
    - `#1` : regular expression(this value will be evaluated by Go's regexp package)
    - `#2` : callback function
        - `m(table)` : captured groups as a list of strings.
        - `e(object)` : message event object:
            - `target(string)`
            - `from(string)`
            - `message(string)`
            - `raw(object)`: underlying procotol specific object
- 4. adds a callback for procotol specific events.
    - `#1` : event name
    - `#2` : callback function
        - `e(object)` : procotol specific event object
- 5. calls underlying procotol specific client methods.
- 6. creates a new goroutine and sends a message to it.
- 7. starts main goroutine.
    - `#1` : callback function that will be called when messages are sent by worker goroutines.  The callback function that will be called when messages are sent by main gorougine.
- 8. responds to the message from other goroutines.
- 9. a function that will be executed in worker goroutines.
- 10. sends a message to the main goroutine.
- 11. a function that will be executed when http requests are arrived.
- 12. sends a message to the main goroutine and receives a result from the main goroutine.

## IRC 

golbot uses [go-ircevent](https://github.com/thoj/go-ircevent) as an IRC client, [GopherLua](https://github.com/yuin/gopher-lua) as a Lua script runtime, and [gopher-luar](https://github.com/layeh/gopher-luar) as a data converter between Go and Lua.

- `golbot init irc` generates a default lua config for IRC bots.
- `golbot.newbot` creates new `*irc.Connection` (in `go-ircevent`)  object wrapped by gopher-luar, so `bot.raw` has same methods as `*irc.Connection` .
- Protocol specific event object has same method as `*irc.Event`
- Protocol specific event names are same as first argument of `*irc.Connection#AddCallback` .
- Protocol specific options for `golbot.newbot` are:
    - `conn(string)` : `"host:port,#channel,#channel..."`
    - `useTLS(bool)`
    - `password(string)`
- There is no official "mention" functionallity in IRC. `@nick`, `:nick` and `\nick` are treated as a mention in `respond`.

## Slack

golbot uses [slack](https://github.com/nlopes/slack) as a Slack client.

- `golbot init slack` generates a default lua config for Slack bots.
- `golbot.newbot` creates new `*slack.RTM` (in `slack`)  object wrapped by gopher-luar, so `bot.raw` has same methods as `*slack.RTM` .
- Protocol specific event object has same method as `*slack.*Event`. Events are listed near [line 349 of websocket_managed_conn.go](https://github.com/nlopes/slack/blob/master/websocket_managed_conn.go#L349) .
- Protocol specific event names are same as [message type of Slack API](https://api.slack.com/events/message)
- Protocol specific options for `golbot.newbot` are:
    - `token(string)` : Bot token

## Hipchat

golbot uses [hipchat](https://github.com/daneharrigan/hipchat) as a Hipchat XMPP client.

- `golbot init hipchat` generates a default lua config for Hipchat bots.
- `golbot.newbot` creates new `*hipchat.Client` (in `hipchat`)  object wrapped by gopher-luar, so `bot.raw` has same methods as `*hipchat.Client` .
- Protocol specific event object has same method as `*hipchat.Message`.
- Protocol specific event names are
    - `"message"`
- Protocol specific options for `golbot.newbot` are:
    - `user(string)` : Hipchat XMPP user name. You can find it on the Hipchat account page(https://yourdomain.hipchat.com/account/xmpp)
    - `password(string)` : password
    - `host(string)` : hostname like `"chat.hipchat.com"`.
    - `conf(string)` : conf hostname like `"conf.hipchat.com"`.
    - `room_jids(list of string)`: XMPP JIDs like `{"111111_xxxxx@conf.hipchat.com", "111111_yyyy@conf.hipchat.com"}`

As described above golbot can also act as an HTTP(S) server, so you can use golbot as [your own integration](https://blog.hipchat.com/2015/02/11/build-your-own-integration-with-hipchat/) for Hipchat.


```lua
local golbot = require("golbot")
local json = require("json")
local requests = require("requests")

function main()
  golbot.newbot("Null", { http = "0.0.0.0:6669" }):serve(function() end)
end

local headers = {
  {"Content-Type", "application/json; charset=utf-8"}
}

function http(r)
  if r.method == "POST" and r.URL.path == "/webhook" then
    local data = assert(json.decode(r:readbody()))
    local message = data.item.message.message
    local user = data.item.message.from.name
    local room = data.item.room.name

    local ret = {
      message = "hello! from webhook",
      message_format = "html"
    }

    return 200, headers, json.encode(ret)
  end
  return 400, headers, json.encode({result="not found"})
end
```

## RocketChat

golbot uses [gorocket](github.com/detached/gorocket) as a RocketChat client.

- `golbot init rocket` generates a default lua config for RocketChat bots.
- `golbot.newbot` creates new `*realtime.Client` (in `gorocket`)  object wrapped by gopher-luar, so `bot.raw` has same methods as `*realtime.Client` .
- Protocol specific event object has same method as `*apl.Message`. 
- Protocol specific options for `golbot.newbot` are:
    - `url(string)` : RocketChat url
    - `name(string)` : User name
    - `email(string)` : User email
    - `password(string)` : User password
    - `channnels(string)` : Comma-separated channels to join such as "general,releasejobs" .


## Logging

golbot is integrated with [seelog](https://github.com/cihub/seelog) . `golbot.newlog(tbl)` creates a new logger that has a `printf` method.

```lua
  log = golbot.newlog(conf)
  log:printf("[info] msg")
```

First word surrounded by `[]` is a log level in seelog. Rest are actual messages.

## Working with non-UTF8 servers

`charset` module provides character set conversion functions.

- `charset.encode(string, charset)` : converts `string` from utf-8 to `charset` .
- `charset.decode(string, charset)` : converts `string` from `charset` to utf-8 .

Examples:

```lua
bot:say("#ch", charset.encode("こんにちわ", "iso-2022-jp"))

bot:respond(".*", function(m, e)
  bot.log:printf("%s", charset.decode(e.message, "iso-2022-jp"))
end)
```

## Groutine model

golbot consists of multiple goroutines.

- main goroutine : connects and communicates with chat servers.
- worker goroutines : typically execute function may takes a long time.
- http goroutines : handle REST API requests.

Consider, for example, the case of deploying web applications when receives a special message.

```lua
  bot:respond("do_deploy", function(m, e)
     do_deploy()
  end)
```

`do_deploy()` may takes a minute, main goroutine is locked all the time. For avoiding this, `do_deploy` should be executed in worker goroutines via messaging(internally, golbot uses Go channels).

```lua
function main()
  -- blah blah
  bot:respond("do_deploy", function(m, e)
     goworker({ch=e.target, message="do_deploy"})
     bot:say(e.target, "Your deploy request is accepted. Please wait a minute.")
  end)

  bot:serve(function(msg)
    bot:say(msg.ch, msg.message)
  end)
end

function worker(msg)
  if msg.message == "do_deploy" then
    do_deploy()
    notifymain({ch=msg.ch, message="do_deploy successfully completed!"})
  end
end
```

`golbot` provides functions that simplify communications between goroutines through channels.

- goworker(msg:table) : creates a new goroutine and sends the `msg` to it.
- notifymain(msg:table) : sends the `msg` to the main goroutine.
- requestmain(msg:table) : sends the `msg` to the main goroutine and receives a result from the main goroutine.
- respond(requestmsg:table, result:any) : sends the `result` to the requestor.

## Create REST API

If the `http` global function exists in the `golbot.lua`, REST API feature will be enabled.

```lua
function http(r)
  if r.method == "POST" and r.URL.path == "/say" then
    local msg = json.decode(r:readbody())
    local ok, success = requestmain({type="say", channel=msg.channel, message=msg.message})
    if ok and success then
      return 200,
             {
               {"Content-Type", "application/json; charset=utf-8"}
             },
             json.encode({result="ok"})
    else
      return 406,
             {
               {"Content-Type", "application/json; charset=utf-8"}
             },
             json.encode({result="error"})
    end
  end
  return 400,
         {
           {"Content-Type", "text/plain; charset=utf-8"}
         },
         "NOT FOUND"
end
```

`http` function receives `net/http#Request` object wrapped by `gopher-luar` .

`http` function must return 3 objects:

- HTTP status:number : HTTP status code.
- headers:table : a table contains response headers.
- contents:string : response body.

`http` function will be executed in its own thread. You can use `notify*` and `request*` functions for communicating with other goroutines.

### TLS support

An `https` global function and an `https` option for `golbot.newbot` enable TLS support.

```lua

  local bot = golbot.newbot("IRC", {                       -- 2
      -- blah blah...
    https = {
      addr = "0.0.0.0:7000",
      cert = "server.crt",
      key  = "server.key"
    },
  })

  function https(r)
    -- same as `http`
  end
```

## Make HTTP requests

`requests` module provides some utilities for making HTTP requests.

- `requests.request(opt:table)`
    - `#1` : options
        - `method(string)` : HTTP method, such as `"GET"`. This defaults to `"GET"` .
        - `url(string)` : URL
        - `data(string)` : Request body
        - `param` : List of parameters such as `{name1, value1, name2, value2}`
            - If `method` is `GET`, this value will be sent as query strings.
            - Otherwise, this value will be sent as a form-urlencoded value and `Content-Type: application/x-www-form-urlencoded` header will also be sent.
        - `headers` : List of headers such as `{name1, value1, name2, value2}`
    - Retuens `body(string)` and `response(net/http#Response wrapped by gopher-luar)` if succeeded or `nil` and `error(string)` .
- `requests.json(opt:table)`
    - `#1` : options. Almost same as `requests.request`. The following additional options are available.
        - `json(table)` : This value will be encoded into a json string and sent as a request body.
    - Retuens `obj(table)` and `response(net/http#Response wrapped by gopher-luar)` if succeeded or `nil` and `error(string)` .
    - `Content-type: application/json` header will automatically be added to the request headers.

```lua
body, response = requests.request({
                   method="POST", 
                   url="http://example.com/",
                   headers={"Content-Type", "application/json"},
                   data=json.encode(data) })
-- same as above
obj, response =  requests.json({
                   method="POST", 
                   url="http://example.com/",
                   json=data })
```

## Execute scheduled jobs

A `crons` option for `golbot.newbot` enables cron like scheduled jobs.

```lua
function main()
  golbot.newbot("Null", { 
    http = "0.0.0.0:6669" ,
    crons = {
      { "0 * * * * * ", "job1"}
    }
  }):serve(function() end)
end

function job1()
  print "hello!"
end
```

`golbot` uses [cron](https://godoc.org/github.com/robfig/cron) to implement this functionality, first element of a job can be 'CRON Expression Format' in `cron` .


## Bundled Lua libraries

- [gluare](https://github.com/yuin/gluare>)
- [gopher-json](https://github.com/layeh/gopher-json)
- [gluafs](https://github.com/kohkimakimoto/gluafs)
- [gluash](https://github.com/otm/gluash)

## Author

Yusuke Inuzuka

## License

[BSD License](http://opensource.org/licenses/BSD-2-Clause)

