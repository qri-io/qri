def transform(ds, ctx):
  meta = ds.get_meta()
  if not meta:
    ds.set_body(['no title'])
  else:
    ds.set_body(['title: %s' % meta['title']])
