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
  'format': 'json',
  'qri': 'st:0'
}

assert.eq(ds.set_structure(st), None)
assert.eq(ds.get_structure(), exp)

bd = [[10,20,30]]
bd_obj = {'a': [10,20,30]}

# How the body renders a dataframe
expect_bd = "      0   1   2\n0    10  20  30\n"
expect_bd_obj = "      a\n0    10\n1    20\n2    30\n"

ds.body = bd
assert.eq('%s' % ds.body, expect_bd)

ds.body = bd_obj
assert.eq('%s' % ds.body, expect_bd_obj)