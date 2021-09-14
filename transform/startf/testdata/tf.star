
def transform(ds, ctx):
  print("hello world!")
  ds.set_structure({ 'format': 'json', 'schema': { 'type' : 'array' }})
  ds.body = [[1, 1.5, False, 'a','b','c'],
             [2, 2.3, True,  'd','e','f'],
             [3, 4.7, False, 'g','h','i']]
  return ds
