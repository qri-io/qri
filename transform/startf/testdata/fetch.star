load("http.star", "http")

def download(ctx):
  res = http.get(test_server_url)
  return res.json()['foo']

def transform(ds, ctx):
  ds.set_body(ctx.download)