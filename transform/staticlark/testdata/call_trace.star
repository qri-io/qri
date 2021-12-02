
def get_secret():
  return 3

def get_public():
  return 4

def modify(v):
  return v * 2

def dangerous(s, t):
  print('Leak %d' % s)

def bottom(m, n):
  a = m + 1 # m is dangerous, n is ok
  b = a * 2
  dangerous(b, 1)
  dangerous(n, 2)

def middle(w, x, y, z):
  c = w + x # c is dangerous, so are w and x
  d = c * 3 # d is dangerous
  e = y + z
  f = d - 1 # f is dangerous
  g = e + 1
  bottom(f, g)

def top():
  i = get_secret()
  j = get_public()
  k = modify(i)
  l = modify(j)
  middle(i, j, k, l)

top()
