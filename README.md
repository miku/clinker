# clinker

A command line link checker.

```
$ cat listofurls.txt | clinker
```

## Check links in an SOLR index

The clinker command accepts JSON, so you can transmit more than just a link -
but clinker will need the key under which the URLs are stored.

```
$ solrdump -q "source_id:169" -fl "id,url" -server 10.1.1.1:1234/solr/biblio | clinker -j url
```
