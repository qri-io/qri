load('assert.star', 'assert')
load('dataset.star', 'dataset')

ds = dataset.new()

assert.eq(ds.set_meta("foo", "bar"), None)
assert.eq(ds.get_meta(), {"foo": "bar", "qri": "md:0"})


assert.eq(ds.get_structure(), None)

st = {
  'format' : 'json', 
  'schema' : { 'type' : 'array' } 
}

exp = {
  'schema': {
    'type': 'array'
  },
  'errCount': 0,  
  'format': 'json', 
  'qri': 'st:0'
}

assert.eq(ds.set_structure(st), None)
assert.eq(ds.get_structure(), exp)


bd = [[1,2,3]]
bd_obj = {'a': [1,2,3]}

assert.eq(ds.set_body(bd_obj), None)
assert.eq(ds.set_body(bd), None)
assert.eq(ds.set_body("[[1,2,3]]", parse_as="json"), None)

# TODO - haven't thought through this yet
assert.eq(ds.get_body(), bd)

# csv_ds is a global variable provided by dataset_test.go
# round-tripping csv data through starlark shouldn't have significant effects on the 
# encoded data. whitespace is *not* significant.
# csv data is one of the harder formats, where there header row must be preserved
csv_ds.set_body(csv_ds.get_body())

expect_data = [["foo",1,"true"], ["bar",2,"false"], ["bat",3,"meh"]]
assert.eq(expect_data, csv_ds.get_body())
assert.eq(csv_ds.get_structure()['format'], 'csv')
