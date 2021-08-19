def transform(ds, ctx):
  meta = ds.get_meta()
  if not meta:
    ds.body = [['no title']]
  else:
    ds.body = [['title: %s' % meta['title']]]
