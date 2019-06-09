load("http.star", "http")
load("qri.star", "qri")

def download(ds, ctx):
  res = http.get(test_server_url)
  return res.json()['foo']

def transform(ds, ctx):
  ds.set_body(ctx.download)