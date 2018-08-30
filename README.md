# clinker

A command line link checker using HTTP HEAD (prerelease [linux
binary](https://github.com/miku/clinker/releases/tag/v0.0.0).

```
$ echo '{"url": "http://google.com"}' | clinker | jq .
{
  "link": "http://google.com",
  "status": 200,
  "t": "2018-08-30T14:50:16.574565848+02:00",
  "comment": "HEAD",
  "payload": {
    "url": "http://google.com"
  }
}
```

## Checking links in an SOLR index

Using in combination with [solrdump](https://github.com/ubleipzig/solrdump):

```
$ solrdump -q "source_id:169" -fl "id,url" -server 10.1.1.1:1234/solr/biblio | clinker > report.ndj
```

