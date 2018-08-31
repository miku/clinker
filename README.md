# clinker

A command line link checker (prerelease [linux
binary](https://github.com/miku/clinker/releases)).

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

## Checking links in an SOLR index

Using in combination with [solrdump](https://github.com/ubleipzig/solrdump):

```
$ solrdump -q "source_id:169" -fl "id,url" -server 10.1.1.1:1234/solr/biblio | clinker > report.ndj
```

