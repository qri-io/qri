def transform(ds, ctx):
  body = ds.get_body()
  for row in body:
    row[1] = row[1] + 1
  ds.set_body(body)

