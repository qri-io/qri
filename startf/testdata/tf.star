
def transform(ds, ctx):
  print("hello world!")
  ds.set_structure({ 'format': 'json', 'schema': { 'type' : 'array' }})
  ds.set_body([1, 1.5, False, 'a','b','c', { "a" : 1, "b" : True }, [1,2]])
  return ds