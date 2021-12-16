def get_secret():
  return 3

def get_public():
  return 4

def modify(v):
  return v * 2

def dangerous(s, t):
  print('Leak %d' % s)

def something(a, b, c):
  s = a + 1
  t = b + 2
  m = 0
  if c < 10:
    # c influences control flow, therefore
    # it is considered sensitive
    m = a + 3
  else:
    # TODO: validate that `m` is derived from `a`
    # using linear analysis (incorrectly) would have the `else`
    # branch override the `then` branch. Rather, the two branches
    # need to be unioned
    m = b + 4
  dangerous(m, t)

def top():
  x = get_public()
  y = modify(get_public())
  z = get_secret()
  something(x, y, z)

top()
