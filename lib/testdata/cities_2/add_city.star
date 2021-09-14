ds = dataset.latest()
---
ds.body = ds.body.append([["tokyo", 9200000, 48.5, False]])
dataset.commit(ds)
