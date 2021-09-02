load("http.star", "http")
ds = dataset.latest()
---
res = http.get(test_server_url)
---
ds.body = res.json()['foo']
dataset.commit(ds)
