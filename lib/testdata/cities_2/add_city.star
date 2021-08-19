def transform(ds,ctx):
  body = ds.body
  body = body.append([["tokyo", 9200000, 48.5, False]])
  ds.body = body
