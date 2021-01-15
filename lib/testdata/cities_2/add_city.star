def transform(ds,ctx):
  body = ds.get_body()
  body.append(["tokyo", 9200000, 48.5, False])
  ds.set_body(body)
