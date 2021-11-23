
def get_secret():
  return 3

def modify(v):
  return v * 2

def dangerous(s, t):
  print('Leak %d' % s)

def safe(u):
  print('Okay %d' % u)

def processor(m, n):
  a = get_secret()
  b = m
  c = a + 1
  d = modify(n)
  safe(d)
  safe(c)
  dangerous(b, c)
  dangerous(c, d)
  return c

processor()
