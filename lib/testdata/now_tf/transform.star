load("time.star", "time")

def transform(ds, ctx):
  body = ds.get_body([])
  body.append(str(time.now()))
  ds.set_body(body)