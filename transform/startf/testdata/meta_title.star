ds = dataset.latest()

meta = ds.get_meta()
if not meta:
  ds.body = [['no title']]
else:
  ds.body = [['title: %s' % meta['title']]]

dataset.commit(ds)