# clinker

A command line link checker. The clinker tool allows to run basic link checks
on a large number of links fast.

[![Project Status: Active – The project has reached a stable, usable state and is being actively developed.](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#active)

## Installation

Currently, we provide a [precompiled amd64
binary](https://github.com/miku/clinker/releases). To install on other platforms,
you'll need the [Go toolchain](https://golang.org/doc/install).

```
$ go install github.com/miku/clinker/cmd/clinker@master
```

## Usage

The clinker tool reads input from stdin and emits output to stdout (one JSON
result per line). The input can a list of URLs, one per line:

```
$ echo golang.org | clinker | jq
{
  "link": "http://golang.org",
  "h": {
    "User-Agent": [
      "clinker/0.2.4 (https://git.io/fAC27)"
    ]
  },
  "status": 200,
  "t": "2019-06-11T17:01:30.836445813+02:00",
  "comment": "GET",
  "payload": {
    "url": "http://golang.org"
  },
  "headers": {
    "Alt-Svc": [
      "quic=\":443\"; ma=2592000; v=\"46,44,43,39\""
    ],
    "Content-Type": [
      "text/html; charset=utf-8"
    ],
    "Date": [
      "Tue, 11 Jun 2019 15:02:01 GMT"
    ],
    "Strict-Transport-Security": [
      "max-age=31536000; includeSubDomains; preload"
    ],
    "Vary": [
      "Accept-Encoding"
    ],
    "Via": [
      "1.1 google"
    ]
  }
}
```

Additionally, the clinker tools allows to work with JSON data as input as well.
By default, if will find the URL in the *url* field (you can override the field
name with the -j flag).

This way, we can take a document and wrap link check information around it. The
original document will be fully preserved under the *payload* key.


```json
$ echo '{"url": "http://ub.uni-leipzig.de"}' | clinker | jq .
{
  "link": "http://ub.uni-leipzig.de",
  "status": 200,
  "t": "2018-08-31T14:14:19.493554196+02:00",
  "comment": "GET",
  "payload": {
    "url": "http://ub.uni-leipzig.de"
  },
  "header": {
    "Connection": [
      "keep-alive"
    ],
    "Content-Length": [
      "57978"
    ],
    "Content-Type": [
      "text/html; charset=utf-8"
    ],
    "Date": [
      "Fri, 31 Aug 2018 12:14:18 GMT"
    ],
    "Server": [
      "nginx"
    ],
    "X-Powered-By": [
      "PHP/5.6.37"
    ]
  }
}
```

Another example:

```json
$ echo '{"url": "http://google.com"}' | clinker | jq .
{
  "link": "http://google.com",
  "status": 200,
  "t": "2018-08-31T14:13:04.162224015+02:00",
  "comment": "GET",
  "payload": {
    "url": "http://google.com"
  },
  "header": {
    "Cache-Control": [
      "private, max-age=0"
    ],
    "Content-Type": [
      "text/html; charset=ISO-8859-1"
    ],
    "Date": [
      "Fri, 31 Aug 2018 12:13:04 GMT"
    ],
    "Expires": [
      "-1"
    ],
    "P3p": [
      "CP=\"This is not a P3P policy! See g.co/p3phelp for more info.\""
    ],
    "Server": [
      "gws"
    ],
    "Set-Cookie": [
      "1P_JAR=2018-08-...",
      "NID=137=GBHDIs_..."
    ],
    "X-Frame-Options": [
      "SAMEORIGIN"
    ],
    "X-Xss-Protection": [
      "1; mode=block"
    ]
  }
}
```

## Flags

```
$ clinker -h
Usage of clinker:
  -H value
        HTTP header to send (repeatable)
  -X string
        HTTP method (default "GET")
  -b    skip invalid input
  -hp string
        use additional header profile
  -j string
        key under which to find the URLs to check (default "url")
  -size int
        batch urls (default 100)
  -ua string
        use a specific user agent (default "clinker/0.2.4 (https://git.io/fAC27)")
  -verbose
        verbose output
  -version
        show version
  -w int
        number of workers (default 4)
```

## Checking links in an SOLR index

Using in combination with [solrdump](https://github.com/ubleipzig/solrdump):

```
$ solrdump -q "source_id:169" -fl "id,url" -server 10.1.1.1:1234/solr/biblio | clinker > report.ndj
```

## Stats

Generate report, run some stats.

```
$ curl -sL https://git.io/vKXFv | clinker -verbose -w 200 > endpoints.ndj
$ jq -rc '.header|keys[]' endpoints.ndj 2> /dev/null | sort | uniq -c | sort -nr | head -50
   4316 Content-Type
   4314 Date
   4133 Server
   2706 Set-Cookie
   2392 Vary
   1304 X-Powered-By
   1254 Cache-Control
    721 Connection
    652 Content-Length
    512 Expires
    338 X-Frame-Options
    310 X-Runtime
    287 X-Content-Type-Options
    260 Pragma
    205 Content-Language
    199 Strict-Transport-Security
    184 Accept-Ranges
    154 X-Xss-Protection
    151 Last-Modified
     97 Via
     95 Age
     87 Link
     81 Etag
     76 Access-Control-Allow-Origin
     70 Cf-Ray
     57 X-Cocoon-Version
     57 Upgrade
     56 Keep-Alive
     52 X-Cache
     39 Referrer-Policy
     38 X-Ua-Compatible
     36 X-Aspnet-Version
     34 X-Varnish
     26 Alt-Svc
     25 Expect-Ct
     23 X-Drupal-Cache
     18 X-Proxy-Cache
     18 X-Generator
     18 X-Cache-Hits
     16 X-Served-By
     16 X-Adblock-Key
     16 Content-Security-Policy
     15 Content-Security-Policy-Report-Only
     14 X-Timer
     14 Access-Control-Allow-Methods
     13 Access-Control-Allow-Headers
     12 X-Request-Id
     12 Content-Encoding
     11 Served-By
     11 Host-Header
```

More examples:

```
$ solrdump -server 10.0.1.1:8085/solr/biblio -fl id,url |
    jq -rc '. as $doc | .url[] | . as $link | {"id": $doc.id, "url": $link}' |
    clinker -verbose -w 128
```
